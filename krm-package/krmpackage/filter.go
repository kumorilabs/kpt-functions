package krmpackage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	orascontext "oras.land/oras-go/pkg/context"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
)

const (
	MediaTypeBase   string = "application/vnd.kumorilabs.kumori"
	MediaTypeKRM    string = MediaTypeBase + ".krm.v1+yaml"
	MediaTypeConfig string = MediaTypeBase + ".config.v1+yaml"

	AnnotationKind                string = "kumori.kumorilabs.io/kind"
	AnnotationAPIVersion          string = "kumori.kumorilabs.io/apiVersion"
	AnnotationMetadataName        string = "kumori.kumorilabs.io/metadata.name"
	AnnotationMetadataNamespace   string = "kumori.kumorilabs.io/metadata.namespace"
	AnnotationMetadataLabels      string = "kumori.kumorilabs.io/metadata.labels"
	AnnotationMetadataAnnotations string = "kumori.kumorilabs.io/metadata.annotations"
	AnnotationPlatform            string = "kumori.kumorilabs.io/platform"
	AnnotationConfigK8S           string = "config.k8s.io"
	AnnotationConfig              string = "config.kubernetes.io"
	AnnotationPath                string = "config.kubernetes.io/path"
	AnnotationConfigInternal      string = "internal.config.kubernetes.io"
	AnnotationPathInternal        string = "internal.config.kubernetes.io/path"

	githubRegistry string = "ghcr.io"
	githubURL      string = "https://github.com/"
)

type KRMPackageFilter struct {
	Config        *KRMPackage
	filterResults []*krmPackageResult
}

type krmPackageResult struct {
	Action     string
	Package    string
	Platform   string
	Digest     string
	FilePath   string
	Name       string
	APIVersion string
	Kind       string
	IsConfig   bool
	ErrorMsg   string
}

func (i *KRMPackageFilter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {

	switch i.Config.Spec.Action {
	case KRMPackageActionPull:
		newItems, err := i.ociPull()
		if err != nil {
			return nil, err
		}

		items = append(items, newItems...)

		merger := filters.MergeFilter{}

		items, err = merger.Filter(items)
		if err != nil {
			return nil, err
		}

	case KRMPackageActionPush:
		f := filters.IsLocalConfig{IncludeLocalConfig: i.Config.Spec.IncludeLocalConfig}
		packageItems, err := f.Filter(items)
		if err != nil {
			return nil, err
		}
		err = i.ociPush(packageItems)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid action used: %v", i.Config.Spec.Action)
	}

	return items, nil
}

func krmToBytes(item *yaml.RNode) (*bytes.Buffer, error) {

	bytes := &bytes.Buffer{}
	writer := kio.ByteWriter{
		Writer:                bytes,
		KeepReaderAnnotations: false}

	err := writer.Write([]*yaml.RNode{item})
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func toMediaType(item *yaml.RNode) (string, error) {
	mediaTypeGVKS := strings.ToLower("_" + item.GetApiVersion() + "_" + item.GetKind())
	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		return "", err
	}
	mediaTypeGVKS = re.ReplaceAllString(mediaTypeGVKS, "-")
	return MediaTypeKRM + mediaTypeGVKS, nil
}

func (i *KRMPackageFilter) ociPush(items []*yaml.RNode) error {
	memoryStore := content.NewMemory()

	var descriptors []ocispec.Descriptor

	for _, item := range items {

		if isKRM(item) {

			pushItem := item.Copy()

			annotations := pushItem.GetAnnotations()

			path := annotations[AnnotationPath]
			if path == "" {
				path = pushItem.GetName() + "-" + strings.ToLower(pushItem.GetKind()) + ".yaml"
			}

			err := removeInternalAnnotations(pushItem)
			if err != nil {
				return err
			}

			bytes, err := krmToBytes(pushItem)
			if err != nil {
				return err

			}

			descriptor, err := memoryStore.Add(path, MediaTypeKRM, bytes.Bytes())
			if err != nil {
				return err
			}

			for k, v := range getFileAnnotations(pushItem) {
				descriptor.Annotations[k] = v
			}

			descriptors = append(descriptors, descriptor)

			result := i.newKrmPackageResult(pushItem)
			result.Digest = string(descriptor.Digest)
			result.FilePath = path
			i.filterResults = append(i.filterResults, result)

		}

	}

	config, configDesc, err := content.GenerateConfig(nil)
	if err != nil {
		return err
	}
	configDesc.MediaType = MediaTypeConfig

	memoryStore.Set(configDesc, config)

	manifest, manifestDesc, err := content.GenerateManifest(&configDesc, getPackageAnnotations(*i.Config), descriptors...)
	if err != nil {
		return err
	}

	err = memoryStore.StoreManifest(i.Config.Spec.Package, manifestDesc, manifest)
	if err != nil {
		return err
	}

	registry, err := content.NewRegistry(content.RegistryOptions{PlainHTTP: false})
	if err != nil {
		return err
	}

	var copyOpts []oras.CopyOpt

	_, err = oras.Copy(orascontext.Background(), memoryStore, i.Config.Spec.Package, registry, "", copyOpts...)
	if err != nil {
		return err
	}
	return nil
}

func (i *KRMPackageFilter) ociPull() ([]*yaml.RNode, error) {
	memoryStore := content.NewMemory()

	registry, err := content.NewRegistry(content.RegistryOptions{PlainHTTP: false})
	if err != nil {
		return nil, err
	}

	var copyOpts []oras.CopyOpt

	copyOpts = append(copyOpts, oras.WithAllowedMediaType(MediaTypeKRM))
	copyOpts = append(copyOpts, oras.WithAllowedMediaType(MediaTypeConfig))

	manifestDescriptor, err := oras.Copy(orascontext.Background(), registry, i.Config.Spec.Package, memoryStore, "", copyOpts...)
	if err != nil {
		return nil, err
	}

	_, manifestContent, _ := memoryStore.Get(manifestDescriptor)

	manifest := ocispec.Manifest{}
	err = json.Unmarshal([]byte(manifestContent), &manifest)
	if err != nil {
		return nil, err
	}

	var items []*yaml.RNode

	for _, descriptor := range manifest.Layers {

		_, content, _ := memoryStore.Get(descriptor)

		reader := kio.ByteReader{Reader: bytes.NewReader(content)}
		newItems, err := reader.Read()
		if err != nil {
			return nil, err
		}
		var newItem *yaml.RNode
		if len(newItems) > 1 {
			return nil, fmt.Errorf("more than one item in krm file layer")
		} else {
			newItem = newItems[0]
		}

		path := descriptor.Annotations[ocispec.AnnotationTitle]

		annotation := newItem.GetAnnotations()
		annotation[AnnotationPathInternal] = path
		annotation[AnnotationPath] = path

		err = newItem.SetAnnotations(annotation)
		if err != nil {
			return nil, err
		}

		items = append(items, newItem)

		result := i.newKrmPackageResult(newItem)
		result.Digest = string(descriptor.Digest)
		result.FilePath = path
		i.filterResults = append(i.filterResults, result)
	}

	return items, nil
}

func removeInternalAnnotations(item *yaml.RNode) error {

	internalAnnotations := kioutil.GetInternalAnnotations(item)
	annotations := item.GetAnnotations()

	for k, _ := range internalAnnotations {
		delete(annotations, k)
	}

	err := item.SetAnnotations(annotations)
	if err != nil {
		return err
	}

	return nil
}

func getPackageAnnotations(config KRMPackage) map[string]string {
	result := make(map[string]string)

	result[ocispec.AnnotationSource] = getImageSourceAnnotation(config)
	result[AnnotationPlatform] = config.Spec.Platform

	return result
}

func getFileAnnotations(item *yaml.RNode) map[string]string {

	result := make(map[string]string)

	for key, value := range item.GetAnnotations() {
		result[AnnotationMetadataAnnotations+"/"+key] = value
	}

	for key, value := range item.GetLabels() {
		result[AnnotationMetadataLabels+"/"+key] = value
	}

	result[AnnotationKind] = item.GetKind()
	result[AnnotationAPIVersion] = item.GetApiVersion()
	result[AnnotationMetadataNamespace] = item.GetNamespace()
	result[AnnotationMetadataName] = item.GetName()

	return result

}

func getImageSourceAnnotation(config KRMPackage) string {
	targetRegistry := getTargetRegistry(config.Spec.Package)

	switch targetRegistry {
	case githubRegistry:
		return replaceRegistryWithSourceURL(targetRegistry, githubURL)
	}

	return ""

}

func getTargetRegistry(target string) string {
	targetRegistry := strings.Split(target, "/")[0]
	return targetRegistry
}

func replaceRegistryWithSourceURL(targetRegistry string, sourceURL string) string {
	return sourceURL + strings.Split(strings.ReplaceAll(targetRegistry, sourceURL+"/", ""), ":")[0]
}

func (i *KRMPackageFilter) Results() (framework.Results, error) {
	var results framework.Results
	if len(i.filterResults) == 0 {
		results = append(results, &framework.Result{
			Message: "no results",
		})
		return results, nil
	}
	for _, packageResult := range i.filterResults {
		var (
			msg      string
			severity framework.Severity
		)
		if packageResult.ErrorMsg != "" {
			msg = fmt.Sprintf("failed to package resources: %s", packageResult.ErrorMsg)
			severity = framework.Error
		} else {
			msg = fmt.Sprintf("%s %s %s %s", packageResult.Digest[7:19], packageResult.Package, packageResult.Kind, packageResult.Name)
			severity = framework.Info
		}

		result := &framework.Result{
			Message:  msg,
			Severity: severity,
		}
		results = append(results, result)
	}
	return results, nil
}

func (i *KRMPackageFilter) newKrmPackageResult(item *yaml.RNode) *krmPackageResult {
	return &krmPackageResult{
		Action:     i.Config.Spec.Action,
		Package:    i.Config.Spec.Package,
		Platform:   i.Config.Spec.Platform,
		APIVersion: item.GetApiVersion(),
		Kind:       item.GetKind(),
		Name:       item.GetName(),
		IsConfig:   isLocalConfig(item),
	}
}
