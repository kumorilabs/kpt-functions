package krmpackage

import (
	"github.com/kumorilabs/kpt-functions/krm-package/krmpackage/filters"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

type KRMPackageProcessor struct{}

func (p *KRMPackageProcessor) Process(resourceList *framework.ResourceList) error {

	filter, err := filters.NewKRMPackageFilter(resourceList.FunctionConfig)
	if err != nil {
		return err
	}

	items, err := filter.Filter(resourceList.Items)
	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  err.Error(),
				Severity: framework.Error,
			},
		}
		return resourceList.Results
	}
	resourceList.Items = items

	results, err := filter.Results()
	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  err.Error(),
				Severity: framework.Error,
			},
		}
		return resourceList.Results
	}
	resourceList.Results = results
	return nil
}
