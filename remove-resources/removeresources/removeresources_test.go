package removeresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type test struct {
	name          string
	input         string
	expectedCount int
	resultCount   int
	errorMsg      string
}

func TestRemoveResources(t *testing.T) {
	// don't really need multiple tests right now but I suspect this fn might
	// get more sophisticated in the future
	var tests = []test{
		{
			name:        "remove-all",
			resultCount: 2,
			input: `
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
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  env: test
  logLevel: debug
`,
			expectedCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err, "kio.ParseAll") {
				t.FailNow()
			}

			fn := &Function{}
			out, err := fn.Filter(input)
			if !assert.NoError(t, err, "Filter") {
				t.FailNow()
			}

			results, err := fn.Results()
			if !assert.NoError(t, err, "Results") {
				t.FailNow()
			}

			assert.Equal(t, test.expectedCount, len(out), "expected count")
			assert.Equal(t, test.resultCount, len(results), "result count")
		})
	}
}
