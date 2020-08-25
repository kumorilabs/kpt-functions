package main

import (
	"os"

	gyaml "github.com/ghodss/yaml"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Spec struct {
	Name   string                 `yaml:"name,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

type ArgoCDConfig struct {
	Spec Spec `yaml:"spec,omitempty"`
}

// Inject strongly-typed yaml into a ConfigMap as a string
// This lets us use Kpt setters in yaml configs that are ultimately used in
// ConfigMaps

func main() {
	functionConfig := &ArgoCDConfig{}
	resourceList := &framework.ResourceList{FunctionConfig: functionConfig}

	cmd := framework.Command(resourceList, func() error {
		for _, item := range resourceList.Items {
			ok, err := isTargetConfigMap(functionConfig.Spec.Name, item)
			if err != nil {
				return err
			}
			if ok {
				for key, config := range functionConfig.Spec.Config {
					b, err := gyaml.Marshal(config)
					if err != nil {
						return err
					}

					c := string(b)
					n := yaml.NewScalarRNode(c)

					err = item.PipeE(
						yaml.Get("data"),
						yaml.FieldSetter{
							Name:  key,
							Value: n,
						},
					)
				}
			}
		}

		return nil
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func isTargetConfigMap(name string, item *yaml.RNode) (bool, error) {
	meta, err := item.GetMeta()
	if err != nil {
		return false, err
	}
	if meta.Kind == "ConfigMap" && meta.APIVersion == "v1" && meta.Name == name {
		return true, nil
	}
	return false, nil
}
