package main

import (
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
	resourceList := &framework.ResourceList{}

	cmd := framework.Command(resourceList, func() error {
		for _, item := range resourceList.Items {
			ns, err := item.Pipe(
				yaml.Lookup("metadata", "namespace"),
			)
			if err != nil {
				return err
			}
			if ns != nil {
				err = item.PipeE(
					yaml.Lookup("metadata"),
					yaml.Clear("namespace"),
				)
			}
		}
		return nil
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
