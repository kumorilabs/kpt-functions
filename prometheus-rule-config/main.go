package main

import (
	"fmt"
	"os"

	gyaml "github.com/ghodss/yaml"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ruleGroup struct {
	Name  string                   `yaml:"name,omitempty"`
	Rules []map[string]interface{} `yaml:"rules,omitempty"`
}

type Spec struct {
	Name  string      `yaml:"name,omitempty"`
	Group []ruleGroup `yaml:"group,omitempty"`
}

type PrometheusRuleConfig struct {
	Spec Spec `yaml:"spec,omitempty"`
}

func main() {
	functionConfig := &PrometheusRuleConfig{}
	resourceList := &framework.ResourceList{FunctionConfig: functionConfig}

	cmd := framework.Command(resourceList, func() error {
		for _, item := range resourceList.Items {
			ok, err := isTargetPrometheusRule(functionConfig.Spec.Name, item)
			if err != nil {
				return err
			}
			if ok {
				for _, rg := range functionConfig.Spec.Group {
					rules, err := item.Pipe(
						yaml.LookupCreate(
							yaml.SequenceNode,
							"spec",
							"groups",
							fmt.Sprintf("[name=%s]", rg.Name),
							"rules",
						),
					)
					if err != nil {
						return err
					}

					for _, rule := range rg.Rules {
						alertName, ok := rule["alert"]
						if !ok {
							// we only know how to handle alert rules right now
							continue
						}

						var r *yaml.Node
						_, ok = rule["disabled"]
						if !ok {
							b, err := gyaml.Marshal(rule)
							if err != nil {
								return err
							}

							rn, err := yaml.Parse(string(b))
							if err != nil {
								return err
							}
							r = rn.YNode()
						}

						if err := rules.PipeE(
							yaml.ElementSetter{
								Key:     "alert",
								Value:   alertName.(string),
								Element: r,
							},
						); err != nil {
							return err
						}
					}
				}
			}
		}

		return nil
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func isTargetPrometheusRule(name string, item *yaml.RNode) (bool, error) {
	meta, err := item.GetMeta()
	if err != nil {
		return false, err
	}
	if meta.Kind == "PrometheusRule" && meta.APIVersion == "monitoring.coreos.com/v1" && meta.Name == name {
		return true, nil
	}
	return false, nil
}
