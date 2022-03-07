package krmpackage

import (
	"fmt"
	"strings"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	KRMPackageKind          string = "KRMPackage"
	KRMPackageAPIVersion    string = "fn.kumorilabs.io/v1alpha1"
	KRMPackageActionPush    string = "push"
	KRMPackageActionPull    string = "pull"
	KRMPackageActionDefault string = KRMPackageActionPull
	KRMPackagePlatformEKS   string = "eks"
	KRMPackagePlatformAKS   string = "aks"
	KRMPackagePlatformGKE   string = "gke"
)

var (
	KRMPackageActions   []string = []string{KRMPackageActionPush, KRMPackageActionPull}
	KRMPackagePlatforms []string = []string{KRMPackagePlatformEKS, KRMPackagePlatformAKS, KRMPackagePlatformGKE}
)

type KRMPackage struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Spec               KRMPackageSpec `yaml:"spec,omitempty"`
}

type KRMPackageSpec struct {
	Action             string `yaml:"action,omitempty"`
	Package            string `yaml:"package,omitempty"`
	Platform           string `yaml:"platform,omitempty"`
	IncludeLocalConfig bool   `yaml:"includeLocalConfig,omitempty"`
}

func validateStrings(key string, value string, posibleValues []string) error {
	for _, p := range posibleValues {
		if p == value {
			return nil
		}
	}

	return fmt.Errorf(KRMPackageKind + " resource [" + key + "] key is invalid. Current value: [" + value + "]. Valid values are: " + strings.Join(posibleValues, ", "))

}

func validateStringEmpty(key string, value string) error {
	if value == "" {
		return fmt.Errorf(KRMPackageKind + " resource is missing [" + key + "] key")
	}
	return nil
}

func (i *KRMPackage) Validate() error {

	err := validateStringEmpty("action", i.Spec.Action)
	if err != nil {
		return err
	}
	err = validateStrings("action", i.Spec.Action, KRMPackageActions)
	if err != nil {
		return err
	}

	err = validateStringEmpty("platform", i.Spec.Platform)
	if err != nil {
		return err
	}
	err = validateStrings("platform", i.Spec.Platform, KRMPackagePlatforms)
	if err != nil {
		return err
	}

	err = validateStringEmpty("package", i.Spec.Package)
	if err != nil {
		return err
	}

	return nil
}

func (i *KRMPackage) Default() error {
	if i.Spec.Action == "" {
		i.Spec.Action = KRMPackageActionDefault
	}

	return nil
}
