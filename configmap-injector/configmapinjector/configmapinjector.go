package configmapinjector

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	apiVersionInject    = "fn.kumorilabs.io/v1alpha1"
	apiVersionConfigMap = "v1"
	kindInject          = "ConfigMapInject"
	kindConfigMap       = "ConfigMap"

	configMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`
)

type ConfigMapInjector struct{}

func (i *ConfigMapInjector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	injectSelector := framework.Selector{
		Kinds:       []string{kindInject},
		APIVersions: []string{apiVersionInject},
	}

	injects, err := injectSelector.Filter(items)
	if err != nil {
		return items, err
	}
	injectMap := map[*yaml.RNode]bool{}
	for _, inject := range injects {
		injectMap[inject] = false
	}

	isConfigMap := framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
		return node.GetKind() == kindConfigMap &&
			node.GetApiVersion() == apiVersionConfigMap
	})

	isInjectTarget := func(inject *yaml.RNode) framework.ResourceMatcherFunc {
		return framework.MatchAll(
			isConfigMap,
			framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
				return node.GetName() == inject.GetName() &&
					node.GetNamespace() == inject.GetNamespace()
			}),
		).Match
	}

	// look for target configmaps and inject data
	for i, item := range items {
		for inject := range injectMap {
			if isInjectTarget(inject)(item) {
				injectMap[inject] = true
				configMap, err := injectConfigMap(inject, item.Copy())
				if err != nil {
					return items, err
				}
				items[i] = configMap
			}
		}
	}

	// if no injection occurred, generate a new configmap
	for inject, injected := range injectMap {
		if !injected {
			configMap, err := newConfigMap(inject)
			if err != nil {
				return items, err
			}
			configMap, err = injectConfigMap(inject, configMap)
			if err != nil {
				return items, err
			}

			items = append(items, configMap)
		}
	}

	return items, nil
}

func newConfigMap(inject *yaml.RNode) (*yaml.RNode, error) {
	configMap, err := yaml.Parse(configMapTemplate)
	if err != nil {
		return nil, err
	}
	configMap.SetName(inject.GetName())
	configMap.SetNamespace(inject.GetNamespace())
	configMap.SetLabels(inject.GetLabels())

	annotations := inject.GetAnnotations()
	delete(annotations, konfig.IgnoredByKustomizeAnnotation)
	configMap.SetAnnotations(annotations)

	return configMap, nil
}

func injectConfigMap(inject *yaml.RNode, configMap *yaml.RNode) (*yaml.RNode, error) {
	data, err := inject.GetFieldValue("data")
	if err != nil {
		return configMap, err
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return configMap, fmt.Errorf(
			"data must be a map[string]interface, got %T",
			data,
		)
	}
	transformed := map[string]string{}
	for key, val := range dataMap {
		yml, err := yaml.Marshal(val)
		if err != nil {
			return configMap, err
		}
		transformed[key] = string(yml)
	}

	cmdata := configMap.GetDataMap()
	for key, val := range transformed {
		cmdata[key] = val
	}
	configMap.SetDataMap(cmdata)
	return configMap, nil
}
