package krmpackage

import (
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type SetPathFilter struct {
	Path string
}

func (i *SetPathFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	return kioutil.Map(items, i.setPath)
}

func (i *SetPathFilter) setPath(item *yaml.RNode) (*yaml.RNode, error) {
	annotations := item.GetAnnotations()

	file := filepath.Base(annotations[AnnotationPathInternal])

	path := i.Path + "/" + file
	annotations[AnnotationPathInternal] = path
	annotations[AnnotationPath] = path

	err := item.SetAnnotations(annotations)
	if err != nil {
		return nil, err
	}

	return item, nil
}
