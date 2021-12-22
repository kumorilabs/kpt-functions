package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var key, value string

func main() {
	fn := func(resourceList *framework.ResourceList) error {
		for i := range resourceList.Items {
			item := resourceList.Items[i]

			meta, err := item.GetMeta()
			if err != nil {
				return err
			}
			if len(meta.Name) > 0 {
				if err := item.PipeE(yaml.SetAnnotation(key, value)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	cmd := command.Build(framework.ResourceListProcessorFunc(fn), command.StandaloneEnabled, false)
	cmd.Flags().StringVar(&key, "key", "", "flag key")
	cmd.Flags().StringVar(&value, "value", "", "flag value")

	if key == "" {
		key = "value"
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
