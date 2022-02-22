package main

import (
	"os"

	"github.com/kumorilabs/kpt-functions/pomerium-policy/pomeriumpolicy"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
	p := PomeriumPolicyProcessor{}
	cmd := command.Build(&p, command.StandaloneEnabled, false)

	cmd.Short = "Inject pomerium policy into Ingress resources"
	cmd.Long = "Author pomerium policy and inject it as an annotation into Ingress Resources"

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type PomeriumPolicyProcessor struct{}

func (p *PomeriumPolicyProcessor) discoverFunctionConfig(resourceList *framework.ResourceList) (*yaml.RNode, error) {
	// currently do not handle multiple PomeriumPolicy resources in the input
	// list. The first one found will be used. This is not great.
	fnconfigs, err := pomeriumpolicy.FunctionConfigSelector.Filter(resourceList.Items)
	if err != nil {
		return nil, err
	}
	if len(fnconfigs) > 0 {
		return fnconfigs[0], nil
	}
	return nil, nil
}

func (p *PomeriumPolicyProcessor) Process(resourceList *framework.ResourceList) error {
	var err error

	// if a function config is not provided by the framework,
	// look for them in the input items
	// this will only work for the PomeriumPolicy kind, not ConfigMaps
	// if a function config is provided by the framework AND one or more
	// function configs are in the input items, we still only process the
	// fnconfig provided by the framework b/c we are assuming the consumer
	// intentionally wants to only process the specified function config
	// If you use the discovery approach (nil resourceList.functionConfig),
	// then you can use other kpt functions against the PomeriumPolicy
	// resources. If you use an explicit functionConfig, it is considered a
	// meta resource and excluded from the input list.

	fnconfig := resourceList.FunctionConfig
	if fnconfig == nil {
		fnconfig, err = p.discoverFunctionConfig(resourceList)
		if err != nil {
			return err
		}
	}

	fn, err := pomeriumpolicy.New(fnconfig)
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
