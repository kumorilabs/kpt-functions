package main

import (
	"log"
	"os"
	"strings"

	gyaml "github.com/ghodss/yaml"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO: DEPRECATED: no longer used. remove

type Data struct {
	OIDC OIDC `yaml:"oidc,omitempty" json:"oidc,omitempty"`
}

type OIDC struct {
	Name                   string   `yaml:"name,omitempty" json:"name,omitempty"`
	Issuer                 string   `yaml:"issuer,omitempty" json:"issuer,omitempty"`
	ClientID               string   `yaml:"clientID,omitempty" json:"clientID,omitempty"`
	ClientSecret           string   `yaml:"clientSecret,omitempty" json:"clientSecret,omitempty"`
	CLIClientID            string   `yaml:"cliClientID,omitempty" json:"cliClientID,omitempty"`
	RequestedScopes        []string `yaml:"requestedScopes,omitempty" json:"requestedScopes,omitempty"`
	RequestedIDTokenClaims string   `yaml:"requestedIDTokenClaims,omitempty" json:"requestedIDTokenClaims,omitempty"`
}

type ConfigMap struct {
	Data Data `yaml:"data,omitempty"`
}

func main() {
	functionConfig := &ConfigMap{}
	resourceList := &framework.ResourceList{FunctionConfig: functionConfig}

	cmd := framework.Command(resourceList, func() error {
		for _, item := range Named(ConfigMaps(resourceList.Items), "argocd-cm") {
			config, err := oidcConfig(functionConfig.Data.OIDC)
			if err != nil {
				return err
			}

			if config != "" {
				_, err = item.Pipe(
					yaml.LookupCreate(yaml.ScalarNode, "data"),
					yaml.SetField("oidc.config", yaml.NewScalarRNode(config)),
				)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func ConfigMaps(items []*yaml.RNode) []*yaml.RNode {
	return filter(items, func(y yaml.ResourceMeta) bool {
		return y.Kind == "ConfigMap"
	})
}

func Named(items []*yaml.RNode, name string) []*yaml.RNode {
	return filter(items, func(y yaml.ResourceMeta) bool {
		return y.Name == name
	})
}

func filter(items []*yaml.RNode, p func(y yaml.ResourceMeta) bool) []*yaml.RNode {
	var out []*yaml.RNode
	for _, item := range items {
		meta, err := item.GetMeta()
		if err != nil {
			itemstr, serr := item.String()
			if serr != nil {
				log.Printf("unable to get meta for %+v (%v)", item, serr)
			} else {
				log.Printf("unable to get meta for %s: %v", itemstr, err)
			}
		}
		if p(meta) {
			out = append(out, item)
		}
	}
	return out
}

func oidcConfig(oidc OIDC) (string, error) {
	switch oidc.Name {
	case "google":
		if oidc.Issuer == "" {
			oidc.Issuer = "https://accounts.google.com"
		}
		if oidc.ClientSecret == "" {
			oidc.ClientSecret = "$oidc.google.clientSecret"
		}
		if len(oidc.RequestedScopes) == 0 {
			oidc.RequestedScopes = []string{"openid", "profile", "email"}
		}
	default:
		log.Printf("unsupported provider name: %q", oidc.Name)
		return "", nil
	}

	b, err := gyaml.Marshal(oidc)
	if err != nil {
		return "", err
	}

	c := string(b)
	if strings.TrimSuffix(c, "\n") == "{}" {
		return "", nil
	}

	return c, nil
}
