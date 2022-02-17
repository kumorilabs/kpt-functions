package v1alpha1

import (
	"encoding/json"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	KRMPackageKind = "KRMPackage"
)

type KRMPackage struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	KRMPackageSpec     KRMPackageSpec `yaml:"spec,omitempty"`
}

type KRMPackageSpec struct {
}

func NewKRMPackage(functionConfig *kyaml.RNode) (*KRMPackage, error) {
	n := &KRMPackage{}
	return decode(functionConfig, n)
}

// Decode decodes the input yaml node into KRMPackage struct
func decode(functionConfig *kyaml.RNode, krmPackage *KRMPackage) (*KRMPackage, error) {
	j, err := functionConfig.MarshalJSON()
	if err != nil {
		return nil, err
	}
	json.Unmarshal(j, krmPackage)
	return krmPackage, nil
}
