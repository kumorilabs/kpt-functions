package configmapinjector

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	apiVersionInjector  = "fn.kumorilabs.io/v1alpha1"
	apiVersionConfigMap = "v1"
	kindInject          = "ConfigMapInject"
	kindTemplate        = "ConfigMapTemplate"
	kindConfigMap       = "ConfigMap"
	configMapTemplate   = `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`
)

type injector func(source, target *yaml.RNode) (*yaml.RNode, error)

type injectResult struct {
	Source   *yaml.RNode
	Target   *yaml.RNode
	Keys     []string
	ErrorMsg string
}

type ConfigMapInjector struct {
	injectResults []*injectResult
}

func (i *ConfigMapInjector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	injectors := map[string]injector{
		kindInject:   i.injectConfigMap,
		kindTemplate: i.templateConfigMap,
	}
	var err error
	for kind, injector := range injectors {
		items, err = i.inject(items, kindSelector(kind), injector)
		if err != nil {
			return items, err
		}
	}
	return items, nil
}

func (i *ConfigMapInjector) Results() (framework.Results, error) {
	var results framework.Results
	if len(i.injectResults) == 0 {
		results = append(results, &framework.Result{
			Message: "no injections",
		})
		return results, nil
	}
	for _, injectResult := range i.injectResults {
		var (
			msg        string
			severity   framework.Severity
			sourceName = fmt.Sprintf("%s %s", injectResult.Source.GetKind(), injectResult.Source.GetName())
			targetName = injectResult.Target.GetName()
		)
		if injectResult.ErrorMsg != "" {
			msg = fmt.Sprintf("%s failed to inject ConfigMap: %s", sourceName, injectResult.ErrorMsg)
			severity = framework.Error
		} else {
			msg = fmt.Sprintf("%s -> %s with keys: %v", sourceName, targetName, injectResult.Keys)
			severity = framework.Info
		}

		result := &framework.Result{
			Message:  msg,
			Severity: severity,
			Field: &framework.Field{
				Path: strings.Join(injectResult.Target.FieldPath(), "."),
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

func (i *ConfigMapInjector) inject(items []*yaml.RNode, selector framework.Selector, injector injector) ([]*yaml.RNode, error) {
	sources, err := selector.Filter(items)
	if err != nil {
		return items, err
	}
	sourceMap := map[*yaml.RNode]bool{}
	for _, source := range sources {
		sourceMap[source] = false
	}

	isConfigMap := framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
		return node.GetKind() == kindConfigMap &&
			node.GetApiVersion() == apiVersionConfigMap
	})

	isTarget := func(inject *yaml.RNode) framework.ResourceMatcherFunc {
		return framework.MatchAll(
			isConfigMap,
			framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
				return node.GetName() == inject.GetName() &&
					node.GetNamespace() == inject.GetNamespace()
			}),
		).Match
	}

	// look for target configmaps and inject data
	for i, item := range items {
		for source := range sourceMap {
			if isTarget(source)(item) {
				sourceMap[source] = true
				configMap, err := injector(source, item.Copy())
				if err != nil {
					return items, err
				}
				items[i] = configMap
			}
		}
	}

	// if no injection occurred, generate a new configmap
	for source, injected := range sourceMap {
		if !injected {
			configMap, err := newConfigMap(source)
			if err != nil {
				return items, err
			}
			configMap, err = injector(source, configMap)
			if err != nil {
				return items, err
			}

			items = append(items, configMap)
		}
	}

	return items, nil
}

func kindSelector(kind string) framework.Selector {
	return framework.Selector{
		Kinds:       []string{kind},
		APIVersions: []string{apiVersionInjector},
	}
}

func newConfigMap(inject *yaml.RNode) (*yaml.RNode, error) {
	configMap, err := yaml.Parse(configMapTemplate)
	if err != nil {
		return nil, err
	}
	configMap.SetName(inject.GetName())
	configMap.SetNamespace(inject.GetNamespace())
	configMap.SetLabels(inject.GetLabels())

	annotations := inject.GetAnnotations()
	delete(annotations, konfig.IgnoredByKustomizeAnnotation)
	configMap.SetAnnotations(annotations)

	return configMap, nil
}

func (i *ConfigMapInjector) injectConfigMap(source *yaml.RNode, configMap *yaml.RNode) (*yaml.RNode, error) {
	result := newInjectResult(source, configMap)
	defer func() {
		i.injectResults = append(i.injectResults, result)
	}()

	data, err := source.GetFieldValue("data")
	if err != nil {
		return configMap, err
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		err = fmt.Errorf(
			"data must be a map[string]interface, got %T",
			data,
		)
		result.ErrorMsg = err.Error()
		return configMap, err
	}
	transformed := map[string]string{}
	for key, val := range dataMap {
		yml, err := yaml.Marshal(val)
		if err != nil {
			result.ErrorMsg = err.Error()
			return configMap, err
		}
		transformed[key] = string(yml)
	}

	cmdata := configMap.GetDataMap()
	for key, val := range transformed {
		cmdata[key] = val
		result.Keys = append(result.Keys, key)
	}
	configMap.SetDataMap(cmdata)
	return configMap, nil
}

func (i *ConfigMapInjector) templateConfigMap(source *yaml.RNode, configMap *yaml.RNode) (*yaml.RNode, error) {
	result := newInjectResult(source, configMap)
	defer func() {
		i.injectResults = append(i.injectResults, result)
	}()

	data := source.GetDataMap()

	rawvals, err := source.GetFieldValue("values")
	if err != nil {
		return configMap, err
	}
	values, ok := rawvals.(map[string]interface{})
	if !ok {
		err = fmt.Errorf(
			"values must be a map[string]interface{}, got %T",
			rawvals,
		)
		result.ErrorMsg = err.Error()
		return configMap, err
	}

	rendered := map[string]string{}
	for key, val := range data {
		tmpl, err := template.New(key).Option("missingkey=error").Parse(val)
		if err != nil {
			return configMap, err
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, values)
		if err != nil {
			result.ErrorMsg = err.Error()
			return configMap, err
		}
		rendered[key] = buf.String()
	}

	cmdata := configMap.GetDataMap()
	for key, val := range rendered {
		cmdata[key] = val
		result.Keys = append(result.Keys, key)
	}
	configMap.SetDataMap(cmdata)
	return configMap, nil
}

func newInjectResult(source, target *yaml.RNode) *injectResult {
	return &injectResult{
		Source: source,
		Target: target,
		Keys:   []string{},
	}
}
