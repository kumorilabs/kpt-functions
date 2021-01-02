package main

import (
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ConfigMap struct {
	Data Data `yaml:"data,omitempty"`
}

type Data struct {
	Annotations []string `yaml:"annotations,omitempty"`
}

func main() {
	functionConfig := &ConfigMap{}
	resourceList := &framework.ResourceList{FunctionConfig: functionConfig}

	cmd := framework.Command(resourceList, func() error {
		for _, item := range resourceList.Items {
			for _, a := range functionConfig.Data.Annotations {
				err := item.PipeE(
					yaml.ClearAnnotation(a),
				)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
