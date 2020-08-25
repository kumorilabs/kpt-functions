package main

import (
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var key, value string

func main() {
	resourceList := &framework.ResourceList{}
	cmd := framework.Command(resourceList, func() error {
		// cmd.Execute() will parse the ResourceList.functionConfig into cmd.Flags from
		// the ResourceList.functionConfig.data field.
		for i := range resourceList.Items {
			// modify the resources using the kyaml/yaml library:
			// https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml
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
	})
	cmd.Flags().StringVar(&key, "key", "", "flag key")
	cmd.Flags().StringVar(&value, "value", "", "flag value")

	if key == "" {
		key = "value"
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
