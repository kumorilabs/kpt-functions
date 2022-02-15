package removeresources

import (
	"strconv"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Function struct {
	removed []*yaml.RNode
}

func (fn *Function) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	// just remove all resources
	// we are relying on kpt Selectors to control what items are passed into the
	// filter
	for _, item := range items {
		fn.removed = append(fn.removed, item)
	}
	return []*yaml.RNode{}, nil
}

func (fn *Function) Results() (framework.Results, error) {
	var results framework.Results
	for _, item := range fn.removed {
		meta, err := item.GetMeta()
		if err != nil {
			return results, err
		}
		id := meta.GetIdentifier()

		result := &framework.Result{
			Message:     "removed",
			Severity:    framework.Info,
			ResourceRef: &id,
		}

		filePath, fileIndex, err := kioutil.GetFileAnnotations(item)
		if err != nil {
			return results, err
		}
		result.File = &framework.File{
			Path: filePath,
		}
		fidx, err := strconv.Atoi(fileIndex)
		if err == nil {
			result.File.Index = fidx
		}

		results = append(results, result)
	}
	return results, nil
}
