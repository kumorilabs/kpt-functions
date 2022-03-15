package krmpackage

import (
	"path/filepath"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type GVKNFileNameFilter struct{}

func (i *GVKNFileNameFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	return kioutil.Map(items, setGVKNFileName)
}

func getGVKN(item *yaml.RNode, separator string) string {
	return item.GetApiVersion() + separator + item.GetKind() + separator + item.GetName()
}

func setGVKNFileName(item *yaml.RNode) (*yaml.RNode, error) {
	replacer := strings.NewReplacer("/", "_", ":", "_", ".", "-")

	fileName := strings.ToLower(replacer.Replace(getGVKN(item, "_")) + ".yaml")

	annotations := item.GetAnnotations()

	for k, v := range annotations {

		if k == kioutil.LegacyPathAnnotation || k == kioutil.PathAnnotation {
			if filepath.Dir(v) == "." {
				annotations[k] = fileName
			} else {
				annotations[k] = filepath.Dir(v) + "/" + fileName
			}
		}

		if k == kioutil.LegacyIndexAnnotation || k == kioutil.IndexAnnotation {
			annotations[k] = strconv.Itoa(0)
		}
	}

	err := item.SetAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	return item, nil
}
