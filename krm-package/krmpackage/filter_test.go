package krmpackage

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

type test struct {
	name        string
	input       string
	context     string
	expected    string
	resultCount int
	errorMsg    string
}

func TestKRMPackage(t *testing.T) {
	var tests = []test{
		{
			name:        "default",
			resultCount: 2,
			context:     "test/upstream",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: KRMPackage
metadata:
  name: test-package
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  package: synax.azurecr.io/test-package/test:0.1.0
  platform: eks
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: gcr.io/google-containers/busybox:1.0.0
        resources:
          requests:
            memory: '32Mi'
            cpu: '100m'
          limits:
            memory: '128Mi'
            cpu: '500m'
        ports:
        - containerPort: 8080
`,

			expected: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: KRMPackage
metadata:
  name: test-package
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  package: synax.azurecr.io/test-package/test:0.1.0
  platform: eks
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: gcr.io/google-containers/busybox:1.0.0
        resources:
          requests:
            memory: '32Mi'
            cpu: '100m'
          limits:
            memory: '128Mi'
            cpu: '500m'
        ports:
        - containerPort: 8080
`,
		},
		{
			name:        "include config",
			resultCount: 3,
			context:     "test/upstream",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: KRMPackage
metadata:
  name: test-package
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  source: test/upstream
  package: synax.azurecr.io/test-package/test:0.1.0
  platform: eks
  includeLocalConfig: true
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: gcr.io/google-containers/busybox:1.0.0
        resources:
          requests:
            memory: '32Mi'
            cpu: '100m'
          limits:
            memory: '128Mi'
            cpu: '500m'
        ports:
        - containerPort: 8080
`,

			expected: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: KRMPackage
metadata:
  name: test-package
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  source: test/upstream
  package: synax.azurecr.io/test-package/test:0.1.0
  platform: eks
  includeLocalConfig: true
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: gcr.io/google-containers/busybox:1.0.0
        resources:
          requests:
            memory: '32Mi'
            cpu: '100m'
          limits:
            memory: '128Mi'
            cpu: '500m'
        ports:
        - containerPort: 8080
`,
		},
	}
	runTests(t, tests)
}

func runTests(t *testing.T, tests []test) {
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

			filter, err := newKRMPackageFilter(inout)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{filter},
				Outputs: []kio.Writer{inout},
			}.Execute()

			if test.errorMsg != "" {
				if !assert.NotNil(t, err, test.name) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errorMsg) {
					t.FailNow()
				}
			}

			if test.errorMsg == "" && !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			// get results
			results, err := filter.Results()
			if !assert.NoError(t, err, test.name, test.name) {
				t.FailNow()
			}
			if !assert.Equal(t, test.resultCount, len(results), test.name) {
				t.FailNow()
			}

			actual, err := ioutil.ReadFile(r.Name())
			if !assert.NoError(t, err, test.name) {
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

func newKRMPackageFilter(reader *kio.LocalPackageReadWriter) (*KRMPackageFilter, error) {
	items, err := reader.Read()
	if err != nil {
		return nil, err
	}

	configFilter := filters.MatchFilter{Kind: KRMPackageKind}

	n, err := configFilter.Filter(items)
	if err != nil || len(n) == 0 {
		return nil, fmt.Errorf("KRMPackage config crd missing")
	}

	config := &KRMPackage{}

	err = framework.LoadFunctionConfig(n[0], config)
	if err != nil {
		return nil, err
	}

	filter := &KRMPackageFilter{Config: config}

	return filter, nil
}
