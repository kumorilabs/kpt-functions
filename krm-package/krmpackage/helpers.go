package krmpackage

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func isKRM(n *yaml.RNode) bool {
	meta, err := n.GetMeta()
	if err != nil {
		return false
	}
	if meta.APIVersion == "" {
		return false
	}
	if meta.Kind == "" {
		return false
	}
	if meta.Name == "" {
		return false
	}
	return true
}

func isLocalConfig(n *yaml.RNode) bool {
	return n.GetAnnotations()["config.kubernetes.io/local-config"] != ""
}
