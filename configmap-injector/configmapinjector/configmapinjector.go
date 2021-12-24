package configmapinjector

import (
	"bytes"
	"fmt"
	"text/template"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	apiVersionInjector  = "fn.kumorilabs.io/v1alpha1"
	apiVersionConfigMap = "v1"
	kindInject          = "ConfigMapInject"
	kindTemplate        = "ConfigMapTemplate"
	kindConfigMap       = "ConfigMap"
	configMapTemplate   = `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`
)

type ConfigMapInjector struct{}

func (i *ConfigMapInjector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	items, err := i.injectFilter(items)
	if err != nil {
		return items, err
	}

	items, err = i.templateFilter(items)
	if err != nil {
		return items, err
	}
	return items, err
}

func (i *ConfigMapInjector) injectFilter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	injectSelector := framework.Selector{
		Kinds:       []string{kindInject},
		APIVersions: []string{apiVersionInjector},
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

func (i *ConfigMapInjector) templateFilter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	templateSelector := framework.Selector{
		Kinds:       []string{kindTemplate},
		APIVersions: []string{apiVersionInjector},
	}

	templates, err := templateSelector.Filter(items)
	if err != nil {
		return items, err
	}
	templateMap := map[*yaml.RNode]bool{}
	for _, tmpl := range templates {
		templateMap[tmpl] = false
	}

	isConfigMap := framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
		return node.GetKind() == kindConfigMap &&
			node.GetApiVersion() == apiVersionConfigMap
	})

	isTemplateTarget := func(template *yaml.RNode) framework.ResourceMatcherFunc {
		return framework.MatchAll(
			isConfigMap,
			framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
				return node.GetName() == template.GetName() &&
					node.GetNamespace() == template.GetNamespace()
			}),
		).Match
	}

	// look for target configmaps and inject rendered template
	for i, item := range items {
		for tmpl := range templateMap {
			if isTemplateTarget(tmpl)(item) {
				templateMap[tmpl] = true
				configMap, err := templateConfigMap(tmpl, item.Copy())
				if err != nil {
					return items, err
				}
				items[i] = configMap
			}
		}
	}

	// if no injection occurred, generate a new configmap
	for tmpl, injected := range templateMap {
		if !injected {
			configMap, err := newConfigMap(tmpl)
			if err != nil {
				return items, err
			}
			configMap, err = templateConfigMap(tmpl, configMap)
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

func templateConfigMap(source *yaml.RNode, configMap *yaml.RNode) (*yaml.RNode, error) {
	data := source.GetDataMap()

	rawvals, err := source.GetFieldValue("values")
	if err != nil {
		return configMap, err
	}
	values, ok := rawvals.(map[string]interface{})
	if !ok {
		return configMap, fmt.Errorf(
			"values must be a map[string]interface{}, got %T",
			rawvals,
		)
	}

	rendered := map[string]string{}
	for key, val := range data {
		tmpl, err := template.New(key).Parse(val)
		if err != nil {
			return configMap, err
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, values)
		if err != nil {
			return configMap, err
		}
		rendered[key] = buf.String()
	}

	cmdata := configMap.GetDataMap()
	for key, val := range rendered {
		cmdata[key] = val
	}
	configMap.SetDataMap(cmdata)
	return configMap, nil
}
