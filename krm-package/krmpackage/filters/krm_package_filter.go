package filters

import (
	"fmt"

	"github.com/kumorilabs/kpt-functions/krm-package/krmpackage/api/v1alpha1"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// type packager func(source, target *yaml.RNode) (*yaml.RNode, error)

type KRMPackageFilter struct {
	config        v1alpha1.KRMPackage
	filterResults []*krmPackageResult
}

func NewKRMPackageFilter(functionConfig *yaml.RNode) (*KRMPackageFilter, error) {
	krmPackage, err := v1alpha1.NewKRMPackage(functionConfig)
	if err != nil {
		return nil, err
	}
	return &KRMPackageFilter{
		config: *krmPackage,
	}, nil
}

type krmPackageResult struct {
	Source   string
	Target   string
	Digest   string
	ErrorMsg string
}

func (i *KRMPackageFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {

	for _, item := range items {

		if isKRM(item) {

			apiVersion := item.GetApiVersion()
			kind := item.GetKind()
			name := item.GetName()

			fmt.Println("Resource read", "apiVersion", apiVersion, "kind", kind, "name", name)
		}

	}

	return items, nil
}

func (i *KRMPackageFilter) Results() (framework.Results, error) {
	var results framework.Results
	if len(i.filterResults) == 0 {
		results = append(results, &framework.Result{
			Message: "no packages",
		})
		return results, nil
	}
	for _, packageResult := range i.filterResults {
		var (
			msg        string
			severity   framework.Severity
			sourceName = packageResult.Source
			targetName = packageResult.Target
		)
		if packageResult.ErrorMsg != "" {
			msg = fmt.Sprintf("%s failed to package resources: %s", sourceName, packageResult.ErrorMsg)
			severity = framework.Error
		} else {
			msg = fmt.Sprintf("faled to copy package from %s to %s", sourceName, targetName)
			severity = framework.Info
		}

		result := &framework.Result{
			Message:  msg,
			Severity: severity,
		}
		results = append(results, result)
	}
	return results, nil
}
