package main

import (
	"fmt"
	"os"

	"github.com/kumorilabs/kpt-functions/pomerium-policy/pomeriumpolicy"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
)

func main() {
	p := PomeriumPolicyProcessor{}
	cmd := command.Build(&p, command.StandaloneEnabled, false)

	cmd.Short = "Inject pomerium policy into Ingress resources"
	cmd.Long = "Author pomerium policy and inject it as an annotation into Ingress Resources"

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type PomeriumPolicyProcessor struct{}

func (p *PomeriumPolicyProcessor) Process(resourceList *framework.ResourceList) error {
	fn, err := pomeriumpolicy.New(resourceList.FunctionConfig)
	if err != nil {
		return err
	}

	items, err := fn.Filter(resourceList.Items)
	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  err.Error(),
				Severity: framework.Error,
			},
		}
	}
	resourceList.Items = items

	results, err := fn.Results()
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
