package main

import (
	"fmt"
	"os"

	"github.com/kumorilabs/kpt-functions/remove-resources/removeresources"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
)

func main() {
	p := RemoveResourcesProcessor{}
	cmd := command.Build(&p, command.StandaloneEnabled, false)

	cmd.Short = "Remove resources"
	cmd.Long = "Remove resources that match selectors"

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type RemoveResourcesProcessor struct{}

func (p *RemoveResourcesProcessor) Process(resourceList *framework.ResourceList) error {
	remover := &removeresources.Function{}

	items, err := remover.Filter(resourceList.Items)
	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  err.Error(),
				Severity: framework.Error,
			},
		}
	}
	resourceList.Items = items

	results, err := remover.Results()
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
