package krmpackage

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type PathFilter struct {
	Path    string
	Exclude bool
}

func (i *PathFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	return kioutil.Map(items, i.filterPath)
}

func (i *PathFilter) filterPath(item *yaml.RNode) (*yaml.RNode, error) {

	annotations := item.GetAnnotations()

	if strings.HasPrefix(annotations[AnnotationPathInternal], i.Path) {
		if i.Exclude {
			return nil, nil
		}
		return item, nil
	} else {
		if i.Exclude {
			return item, nil
		}
		return nil, nil
	}

}
