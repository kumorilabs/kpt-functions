package configmapinjector

import (
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	apiVersionInject    = "fn.kumorilabs.io/v1alpha1"
	apiVersionConfigMap = "v1"
	kindInject          = "ConfigMapInject"
	kindConfigMap       = "ConfigMap"
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

	injectMap := map[*yaml.RNode]bool{}
	for _, inject := range injects {
		injectMap[inject] = false
	}

	for i, item := range items {
		for inject := range injectMap {
			if isInjectTarget(inject)(item) {
				injectMap[inject] = true
				data, err := inject.GetFieldValue("data")
				if err != nil {
					return items, err
				}
				dataMap, ok := data.(map[string]interface{})
				if !ok {
					// TODO: how do we surface the fact that we can't continue with the
					// injection?
					continue
				}
				transformed := map[string]string{}
				for key, val := range dataMap {
					yml, err := yaml.Marshal(val)
					if err != nil {
						return items, err
					}
					transformed[key] = string(yml)
				}

				configMap := item.Copy()
				cmdata := configMap.GetDataMap()
				for key, val := range transformed {
					cmdata[key] = val
				}
				configMap.SetDataMap(cmdata)
				items[i] = configMap
			}
		}
	}

	for inject, injected := range injectMap {
		if !injected {
			const cmTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`
			configMap, err := yaml.Parse(cmTemplate)
			if err != nil {
				return items, err
			}
			configMap.SetName(inject.GetName())
			configMap.SetNamespace(inject.GetNamespace())
			configMap.SetLabels(inject.GetLabels())

			annotations := inject.GetAnnotations()
			delete(annotations, konfig.IgnoredByKustomizeAnnotation)
			configMap.SetAnnotations(annotations)

			data, err := inject.GetFieldValue("data")
			if err != nil {
				return items, err
			}
			dataMap, ok := data.(map[string]interface{})
			if !ok {
				// TODO: how do we surface the fact that we can't continue with the
				// injection?
				continue
			}
			transformed := map[string]string{}
			for key, val := range dataMap {
				yml, err := yaml.Marshal(val)
				if err != nil {
					return items, err
				}
				transformed[key] = string(yml)
			}
			configMap.SetDataMap(transformed)
			items = append(items, configMap)
		}
	}

	return items, nil
}
