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

type injector func(source, target *yaml.RNode) (*yaml.RNode, error)

type ConfigMapInjector struct{}

func (i *ConfigMapInjector) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	injectors := map[string]injector{
		kindInject:   injectConfigMap,
		kindTemplate: templateConfigMap,
	}
	var err error
	for kind, injector := range injectors {
		items, err = i.inject(items, kindSelector(kind), injector)
		if err != nil {
			return items, err
		}
	}
	return items, nil
}

func (i *ConfigMapInjector) inject(items []*yaml.RNode, selector framework.Selector, injector injector) ([]*yaml.RNode, error) {
	sources, err := selector.Filter(items)
	if err != nil {
		return items, err
	}
	sourceMap := map[*yaml.RNode]bool{}
	for _, source := range sources {
		sourceMap[source] = false
	}

	isConfigMap := framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
		return node.GetKind() == kindConfigMap &&
			node.GetApiVersion() == apiVersionConfigMap
	})

	isTarget := func(inject *yaml.RNode) framework.ResourceMatcherFunc {
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
		for source := range sourceMap {
			if isTarget(source)(item) {
				sourceMap[source] = true
				configMap, err := injector(source, item.Copy())
				if err != nil {
					return items, err
				}
				items[i] = configMap
			}
		}
	}

	// if no injection occurred, generate a new configmap
	for source, injected := range sourceMap {
		if !injected {
			configMap, err := newConfigMap(source)
			if err != nil {
				return items, err
			}
			configMap, err = injector(source, configMap)
			if err != nil {
				return items, err
			}

			items = append(items, configMap)
		}
	}

	return items, nil
}

func kindSelector(kind string) framework.Selector {
	return framework.Selector{
		Kinds:       []string{kind},
		APIVersions: []string{apiVersionInjector},
	}
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
		tmpl, err := template.New(key).Option("missingkey=error").Parse(val)
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
