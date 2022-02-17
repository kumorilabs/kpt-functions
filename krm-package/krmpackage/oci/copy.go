package krmpackage

import (
	"bytes"
	"context"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/containerd/containerd/images"
	"github.com/go-logr/logr"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	orascontext "oras.land/oras-go/pkg/context"

	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
)

const (
	mediaTypeBase   string = "application/vnd.kumorilabs.kumori"
	mediaTypeYaml   string = mediaTypeBase + ".file.v1+yaml"
	mediaTypeConfig string = mediaTypeBase + ".config.v1+json"
	githubRegistry  string = "ghcr.io"
	githubURL       string = "https://github.com/"
)

type CopierConfig struct {
	Source string
	Target string
}

type Copier struct {
	config      CopierConfig
	log         logr.Logger
	store       *content.Memory
	descriptors []ocispec.Descriptor
}

func NewCopier(log logr.Logger, copierConfig CopierConfig) *Copier {
	runtime.ErrorHandlers = runtime.ErrorHandlers[1:]
	return &Copier{
		log:         log.WithName("copier"),
		config:      copierConfig,
		descriptors: []ocispec.Descriptor{},
		// we are using a memory store because we read using kio
		store: content.NewMemory(),
	}
}

func (p *Copier) Run(ctx context.Context) error {

	p.readPackage()

	config, configDesc, err := content.GenerateConfig(nil)
	if err != nil {
		return err
	}

	configDesc.MediaType = mediaTypeConfig

	p.store.Set(configDesc, config)

	manifestAnnotations := make(map[string]string)
	manifestAnnotations = addImageSourceAnnotation(p.config.Target, manifestAnnotations)

	manifest, manifestDesc, err := content.GenerateManifest(&configDesc, manifestAnnotations, p.descriptors...)
	if err != nil {
		return err
	}

	err = p.store.StoreManifest(p.config.Target, manifestDesc, manifest)
	if err != nil {
		return err
	}

	registry, err := content.NewRegistry(content.RegistryOptions{PlainHTTP: false})
	if err != nil {
		return err
	}

	var copyOpts []oras.CopyOpt

	copyOpts = append(copyOpts, oras.WithPullCallbackHandler(images.HandlerFunc(p.logProgress)))

	_, err = oras.Copy(orascontext.Background(), p.store, p.config.Target, registry, "", copyOpts...)
	if err != nil {
		return err
	}

	return nil
}

func (p *Copier) readPackage() error {
	// read the resources from output dir
	in := &kio.LocalPackageReader{
		PackagePath:       p.config.Source,
		PreserveSeqIndent: true,
		WrapBareSeqNode:   true,
	}
	out := &bytes.Buffer{}

	err := kio.Pipeline{
		Inputs:  []kio.Reader{in},
		Outputs: []kio.Writer{&kio.ByteWriter{Writer: out}},
	}.Execute()
	if err != nil {
		return err
	}

	// resources, err := kio.FromBytes(out.Bytes())
	// if err != nil {
	// 	return err
	// }
	// for _, resource := range resources {
	// 	// if isKRM(resource) {

	// 	// 	apiVersion := resource.GetApiVersion()
	// 	// 	kind := resource.GetKind()
	// 	// 	name := resource.GetName()

	// 	// 	p.log.Info("Resource read", "apiVersion", apiVersion, "kind", kind, "name", name)
	// 	// }

	// 	// kio.Writer{}

	// 	// desc, err := p.store.Add(resource.GetName(), mediaTypeYaml, resource)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }

	// 	// p.descs = append(p.descs, desc)

	// }
	return nil

}

func (p *Copier) logProgress(ctx context.Context, d ocispec.Descriptor) (children []ocispec.Descriptor, err error) {
	if d.MediaType == mediaTypeYaml {
		file := d.Annotations[ocispec.AnnotationTitle]
		digest := d.Digest.Encoded()[0:12]

		p.log.Info("copying file", "source", p.config.Source, "target", p.config.Target, "digest", digest, "file", file)
	}
	return
}

func addImageSourceAnnotation(target string, m map[string]string) map[string]string {
	targetRegistry := getTargetRegistry(target)

	switch targetRegistry {
	case githubRegistry:
		m[ocispec.AnnotationSource] = replaceRegistryWithSourceURL(targetRegistry, githubURL)
	}

	return m

}

func getTargetRegistry(target string) string {
	targetRegistry := strings.Split(target, "/")[0]
	return targetRegistry
}

func replaceRegistryWithSourceURL(targetRegistry string, sourceURL string) string {
	return sourceURL + strings.Split(strings.ReplaceAll(targetRegistry, sourceURL+"/", ""), ":")[0]
}
