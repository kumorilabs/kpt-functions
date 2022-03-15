package krmpackage

import (
	"fmt"
	"regexp"
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
	Path               string `yaml:"path,omitempty"`
	Package            string `yaml:"package,omitempty"`
	Platform           string `yaml:"platform,omitempty"`
	GVKNFileNames      *bool  `yaml:"gkvnFileNames,omitempty"`
	SingleFileOutput   *bool  `yaml:"singleFileOutput,omitempty"`
	ResourceMerge      *bool  `yaml:"resourceMerge,omitempty"`
	IncludeLocalConfig *bool  `yaml:"includeLocalConfig,omitempty"`
}

func validateStringRegex(key string, value string, regex string, validValuesMessage string) error {

	r, err := regexp.Compile(regex)
	if err != nil {
		return err
	}
	if r.MatchString(value) {
		return nil
	}

	return fmt.Errorf(KRMPackageKind + " resource [" + key + "] key is invalid. Current value: [" + value + "]. " + validValuesMessage)
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

	// err = validateStringEmpty("platform", i.Spec.Platform)
	// if err != nil {
	// 	return err
	// }
	// err = validateStrings("platform", i.Spec.Platform, KRMPackagePlatforms)
	// if err != nil {
	// 	return err
	// }

	if i.Spec.Path != "" {
		err = validateStringRegex("path", i.Spec.Path, "^[a-z0-9]([a-z0-9-]*[a-z0-9])?(/[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$", "Value must be a valid relative path (no slash at the end or beginning), examples: foo/bar, foo")
		if err != nil {
			return err
		}
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

	if i.Spec.ResourceMerge == nil {
		i.Spec.ResourceMerge = newTrue()
	}

	if i.Spec.GVKNFileNames == nil {
		i.Spec.GVKNFileNames = newTrue()
	}

	if i.Spec.IncludeLocalConfig == nil {
		i.Spec.IncludeLocalConfig = newFalse()
	}

	if i.Spec.SingleFileOutput == nil {
		i.Spec.SingleFileOutput = newFalse()
	}

	return nil
}

func newTrue() *bool {
	b := true
	return &b
}

func newFalse() *bool {
	b := false
	return &b
}
