package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestAnnotationFn(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		config   *Config
		expected string
	}{
		{
			name: "annotates a resource",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
`,
			config: &Config{"key", "value"},
			expected: `apiVersion: v1
kind: Service
metadata:
  name: aservice
  annotations:
    key: 'value'
spec:
  ports:
  - port: 8080
`,
		},
		{
			name: "annotates a resource",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: v1
kind: Namespace
metadata:
  name: myns
`,
			config: &Config{"some", "val"},
			expected: `apiVersion: v1
kind: Service
metadata:
  name: aservice
  annotations:
    some: 'val'
spec:
  ports:
  - port: 8080
---
apiVersion: v1
kind: Namespace
metadata:
  name: myns
  annotations:
    some: 'val'
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

			inout := &kio.LocalPackageReadWriter{
				PackagePath: baseDir,
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{kio.FilterFunc(annotate(test.config))},
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
