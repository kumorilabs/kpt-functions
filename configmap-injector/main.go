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
	cmd.Long = "Inject files wrapped in KRM resources into ConfigMap keys"

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type ConfigMapInjectorProcessor struct{}

func (p *ConfigMapInjectorProcessor) Process(resourceList *framework.ResourceList) error {
	injector := &configmapinjector.ConfigMapInjector{}
	var err error
	resourceList.Items, err = injector.Filter(resourceList.Items)
	if err != nil {
		return fmt.Errorf("error injecting configmap: %v", err)
	}
	return nil
}
