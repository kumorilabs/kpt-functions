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
			meta, err := item.GetMeta()
			if err != nil {
				return err
			}

			if meta.Kind == "RoleBinding" || meta.Kind == "ClusterRoleBinding" {
				subjects, err := item.Pipe(
					yaml.Lookup("subjects"),
				)
				if err != nil {
					return err
				}
				err = subjects.VisitElements(func(n *yaml.RNode) error {
					return n.PipeE(yaml.Clear("namespace"))
				})
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
