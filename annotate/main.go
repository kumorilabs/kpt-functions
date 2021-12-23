package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Config struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func main() {
	config := &Config{}

	p := framework.SimpleProcessor{
		Config: config,
		Filter: kio.FilterFunc(annotate(config)),
	}

	cmd := command.Build(p, command.StandaloneEnabled, false)
	cmd.Flags().StringVar(&config.Key, "key", "", "flag key")
	cmd.Flags().StringVar(&config.Value, "value", "", "flag value")

	if config.Key == "" {
		config.Key = "some-annotation"
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func annotate(config *Config) func([]*yaml.RNode) ([]*yaml.RNode, error) {
	return func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range items {
			item := items[i]

			meta, err := item.GetMeta()
			if err != nil {
				return items, err
			}

			if len(meta.Name) > 0 {
				if err := item.PipeE(yaml.SetAnnotation(config.Key, config.Value)); err != nil {
					return items, err
				}
			}

		}
		return items, nil
	}
}
