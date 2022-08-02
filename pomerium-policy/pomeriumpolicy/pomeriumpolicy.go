package pomeriumpolicy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pomerium/pomerium/pkg/policy/parser"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnApiVersion = "fn.kumorilabs.io/v1alpha1"
	fnKind       = "PomeriumPolicy"
	ingressKind  = "Ingress"
)

var (
	ingressApiVersions = []string{
		"networking.k8s.io/v1",
		"extensions/v1beta1",
		"networking.k8s.io/v1beta1",
	}
	FunctionConfigSelector = framework.Selector{
		Kinds:       []string{fnKind},
		APIVersions: []string{fnApiVersion},
	}
)

type injectResult struct {
	Source   *yaml.RNode
	Target   *yaml.RNode
	ErrorMsg string
}

type Function struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`
	Policy            []map[string]interface{} `json:"policy,omitempty"`
	injectResults     []*injectResult
	fnconfig          *yaml.RNode
	validationResult  *framework.Result
}

func New(fnconfig *yaml.RNode) (*Function, error) {
	if fnconfig == nil {
		return nil, errors.New("no functionConfig specified")
	}

	fn := &Function{
		fnconfig: fnconfig,
	}

	meta, err := fnconfig.GetMeta()
	if err != nil {
		return nil, errors.New("unable to get resource meta from functionConfig")
	}
	fn.ResourceMeta = meta

	switch {
	case validGVK(meta, "v1", "ConfigMap"):
		return fn, unmarshalConfig(fn, fnconfig, "data")
	case validGVK(meta, fnApiVersion, fnKind):
		return fn, unmarshalConfig(fn, fnconfig, "")
	default:
		return nil, fmt.Errorf("functionConfig must be a ConfigMap or %s", fnKind)
	}
}

func (fn *Function) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	isIngress := framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
		for _, apiV := range ingressApiVersions {
			if node.GetApiVersion() == apiV && node.GetKind() == ingressKind {
				return true
			}
		}
		return false
	})

	policyjson, err := json.Marshal(fn.Policy)
	if err != nil {
		return items, err
	}

	_, err = parser.ParseJSON(bytes.NewReader(policyjson))
	if err != nil {
		fn.validationResult = fn.validationErrorResult(err)
		return items, nil
	}

	for _, item := range items {
		if isIngress(item) {
			annotations := item.GetAnnotations()
			annotations["ingress.pomerium.io/policy"] = string(policyjson)
			item.SetAnnotations(annotations)
			fn.injectResults = append(fn.injectResults, &injectResult{
				Source: fn.fnconfig,
				Target: item,
			})
		}
	}
	return items, nil
}

func (fn *Function) validationErrorResult(err error) *framework.Result {
	result := &framework.Result{
		Message:  fmt.Sprintf("invalid pomerium policy: %v", err),
		Severity: framework.Error,
		Field: &framework.Field{
			Path: strings.Join(fn.fnconfig.FieldPath(), "."),
		},
		ResourceRef: &yaml.ResourceIdentifier{
			TypeMeta: yaml.TypeMeta{
				APIVersion: fn.fnconfig.GetApiVersion(),
				Kind:       fn.fnconfig.GetKind(),
			},
			NameMeta: yaml.NameMeta{
				Name:      fn.fnconfig.GetName(),
				Namespace: fn.fnconfig.GetNamespace(),
			},
		},
	}
	filePath, fileIndex, err := kioutil.GetFileAnnotations(fn.fnconfig)
	if err == nil {
		result.File = &framework.File{
			Path: filePath,
		}
		fidx, err := strconv.Atoi(fileIndex)
		if err == nil {
			result.File.Index = fidx
		}
	}
	return result
}

func (fn *Function) Results() (framework.Results, error) {
	var results framework.Results

	if fn.validationResult != nil {
		results = append(results, fn.validationResult)
		return results, nil
	}

	if len(fn.injectResults) == 0 {
		results = append(results, &framework.Result{
			Severity: framework.Warning,
			Message:  "no target Ingress resources found",
		})
		return results, nil
	}

	for _, injectResult := range fn.injectResults {
		var (
			msg        string
			severity   framework.Severity
			sourceName = fmt.Sprintf("%s/%s", injectResult.Source.GetKind(), injectResult.Source.GetName())
			targetName = fmt.Sprintf("%s/%s", injectResult.Target.GetKind(), injectResult.Target.GetName())
		)
		if injectResult.ErrorMsg != "" {
			msg = fmt.Sprintf("%s failed to inject policy into %s annotation: %s", sourceName, targetName, injectResult.ErrorMsg)
			severity = framework.Error
		} else {
			msg = fmt.Sprintf("%s injected into %s", sourceName, targetName)
			severity = framework.Info
		}

		result := &framework.Result{
			Message:  msg,
			Severity: severity,
			Field: &framework.Field{
				Path: strings.Join(injectResult.Target.FieldPath(), "."),
			},
			ResourceRef: &yaml.ResourceIdentifier{
				TypeMeta: yaml.TypeMeta{
					APIVersion: injectResult.Target.GetApiVersion(),
					Kind:       injectResult.Target.GetKind(),
				},
				NameMeta: yaml.NameMeta{
					Name:      injectResult.Target.GetName(),
					Namespace: injectResult.Target.GetNamespace(),
				},
			},
		}

		filePath, fileIndex, err := kioutil.GetFileAnnotations(injectResult.Target)
		if err != nil {
			return results, err
		}
		result.File = &framework.File{
			Path: filePath,
		}
		fidx, err := strconv.Atoi(fileIndex)
		if err == nil {
			result.File.Index = fidx
		}

		results = append(results, result)
	}
	return results, nil
}

func validGVK(meta yaml.ResourceMeta, apiVersion, kind string) bool {
	if meta.APIVersion != apiVersion || meta.Kind != kind {
		return false
	}
	return true
}

func unmarshalConfig(fn *Function, rn *yaml.RNode, field string) error {
	node := rn

	if field != "" {
		spec := rn.Field(field)
		if spec == nil {
			return nil
		}
		node = spec.Value
	}

	yamlstr, err := node.String()
	if err != nil {
		return fmt.Errorf("unable to get yaml from functionConfig: %w", err)
	}

	if err := yaml.Unmarshal([]byte(yamlstr), fn); err != nil {
		return fmt.Errorf("unable to unmarshal functionConfig: %w", err)
	}
	return nil
}
