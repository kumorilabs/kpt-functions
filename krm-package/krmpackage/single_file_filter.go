package krmpackage

import (
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type SingleFileFilter struct {
	FileName  string
	fileCount int
}

func (i *SingleFileFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	i.fileCount = 0
	return kioutil.Map(items, i.setFileName)
}

func (i *SingleFileFilter) setFileName(item *yaml.RNode) (*yaml.RNode, error) {

	replacer := strings.NewReplacer("/", "_", ":", "_", ".", "-")
	fileName := replacer.Replace(i.FileName) + ".yaml"

	annotations := item.GetAnnotations()

	for k, _ := range annotations {
		if k == kioutil.LegacyPathAnnotation || k == kioutil.PathAnnotation {
			annotations[k] = fileName
		}

		if k == kioutil.LegacyIndexAnnotation || k == kioutil.LegacyIdAnnotation || k == kioutil.IndexAnnotation {
			annotations[k] = strconv.Itoa(i.fileCount)
		}
	}

	i.fileCount++

	err := item.SetAnnotations(annotations)
	if err != nil {
		return nil, err
	}

	return item, nil
}
