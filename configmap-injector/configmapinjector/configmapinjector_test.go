package configmapinjector

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type test struct {
	name     string
	input    string
	expected string
}

func TestConfigMapInjector(t *testing.T) {
	var tests = []test{
		{
			name: "single injection",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapInject
metadata:
  name: argocd-cm
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  repository.credentials:
  - url: https://github.com/kumorilabs # kpt-set: ${git-base-url}
    passwordSecret:
      key: password
      name: git-reader
    usernameSecret:
      key: username
      name: git-reader
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
  name: argocd-cm
`,
			expected: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapInject
metadata:
  name: argocd-cm
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  repository.credentials:
  - url: https://github.com/kumorilabs # kpt-set: ${git-base-url}
    passwordSecret:
      key: password
      name: git-reader
    usernameSecret:
      key: username
      name: git-reader
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
  name: argocd-cm
data:
  repository.credentials: |
    - passwordSecret:
        key: password
        name: git-reader
      url: https://github.com/kumorilabs
      usernameSecret:
        key: username
        name: git-reader
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			r, err := ioutil.TempFile(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			defer os.Remove(r.Name())
			err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			injector := &ConfigMapInjector{}
			inout := &kio.LocalPackageReadWriter{
				PackagePath: baseDir,
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{injector},
				Outputs: []kio.Writer{inout},
			}.Execute()

			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			actual, err := ioutil.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !assert.Equal(t,
				strings.TrimSpace(test.expected),
				strings.TrimSpace(string(actual))) {
				t.FailNow()
			}
		})
	}
}
