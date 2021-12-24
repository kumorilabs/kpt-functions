package main

import (
	"fmt"
	"os"

	"github.com/kumorilabs/kpt-functions/configmap-injector/configmapinjector"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
)

func main() {
	p := ConfigMapInjectorProcessor{}
	cmd := command.Build(&p, command.StandaloneEnabled, false)

	cmd.Short = "Inject files wrapped in KRM resources into ConfigMap keys"
	cmd.Long = "Inject files or templates wrapped in KRM resources into ConfigMap keys"

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type ConfigMapInjectorProcessor struct{}

func (p *ConfigMapInjectorProcessor) Process(resourceList *framework.ResourceList) error {
	injector := &configmapinjector.ConfigMapInjector{}

	items, err := injector.Filter(resourceList.Items)
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

	results, err := injector.Results()
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
