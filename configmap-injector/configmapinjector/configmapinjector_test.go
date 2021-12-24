package configmapinjector

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type test struct {
	name     string
	input    string
	expected string
}

func TestConfigMapInjectorInject(t *testing.T) {
	var tests = []test{
		{
			name: "single key injection",
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
		{
			name: "multiple key injections",
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
  another-key:
    enabled: false
    name: someval
    num: 87
    val: "45"
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
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
  name: argocd-cm
data:
  another-key: |
    enabled: false
    name: someval
    num: 87
    val: "45"
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
		{
			name: "multiple injections",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapInject
metadata:
  name: cm1
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  someyaml:
    with: values
    andLists:
      - one
      - two
---
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapInject
metadata:
  name: cm2
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  morestuff:
    - map: val
      num: 45
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data: {}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
`,
			expected: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  someyaml: |
    andLists:
    - one
    - two
    with: values
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
data:
  morestuff: |
    - map: val
      num: 45
`,
		},
		{
			name: "merges into existing",
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
  name: argocd-cm
data:
  some.existing: |
    num: 3
    mode: test
  test: val
`,
			expected: `
apiVersion: v1
kind: ConfigMap
metadata:
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
  some.existing: |
    num: 3
    mode: test
  test: val
`,
		},
		{
			name: "generates configmap if it doesn't exist",
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
`,
			expected: `
apiVersion: v1
kind: ConfigMap
metadata:
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

			configMaps := &framework.Selector{
				Kinds:       []string{kindConfigMap},
				APIVersions: []string{apiVersionConfigMap},
			}

			injector := &ConfigMapInjector{}
			inout := &kio.LocalPackageReadWriter{
				PackagePath: baseDir,
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{injector, configMaps},
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

func TestConfigMapInjectorTemplate(t *testing.T) {
	var tests = []test{
		{
			name: "single key template",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapTemplate
metadata:
  name: some-cm
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  config.json: |
    {
      "deployment": {
        "files": {
          "example-resource-file1": {
            "sourceUrl": "{{.s3BaseUrl}}/{{.s3Bucket}}/example-application/example-resource-file1"
          },
          "images/example-resource-file2": {
            "sourceUrl": "{{.s3BaseUrl}}/{{.s3Bucket}}/example-application/images/example-resource-file2"
          },
        }
      },
      "id": "v1",
      "runtime": "python27",
      "threadsafe": true,
    }
values:
  s3BaseUrl: https://my-s3.com # kpt-set: ${s3BaseUrl}
  s3Bucket: my-bucket # kpt-set: ${s3Bucket}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-cm
`,
			expected: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-cm
data:
  config.json: |
    {
      "deployment": {
        "files": {
          "example-resource-file1": {
            "sourceUrl": "https://my-s3.com/my-bucket/example-application/example-resource-file1"
          },
          "images/example-resource-file2": {
            "sourceUrl": "https://my-s3.com/my-bucket/example-application/images/example-resource-file2"
          },
        }
      },
      "id": "v1",
      "runtime": "python27",
      "threadsafe": true,
    }
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

			configMaps := &framework.Selector{
				Kinds:       []string{kindConfigMap},
				APIVersions: []string{apiVersionConfigMap},
			}

			injector := &ConfigMapInjector{}
			inout := &kio.LocalPackageReadWriter{
				PackagePath: baseDir,
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{injector, configMaps},
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
