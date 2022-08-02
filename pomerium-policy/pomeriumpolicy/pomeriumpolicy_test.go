package pomeriumpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNew(t *testing.T) {
	for _, test := range []struct {
		name                string
		fnconfig            string
		errorMsg            string
		expectedActionCount int
	}{
		{
			name:     "nil-functionConfig",
			errorMsg: "no functionConfig specified",
		},
		{
			name: "bad-gvk",
			fnconfig: `
apiVersion: v1
kind: SomeKind
metadata:
  name: policy
`,
			errorMsg: "functionConfig must be a ConfigMap or PomeriumPolicy",
		},
		{
			name: "configmap-no-data",
			fnconfig: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: policy
`,
		},
		{
			name: "configmap-empty-policy",
			fnconfig: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: policy
data:
  policy: []
`,
		},
		{
			name: "configmap-single-action",
			fnconfig: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: policy
data:
  policy:
  - allow:
      and:
      - email:
          is: user@domain.com
`,
			expectedActionCount: 1,
		},
		{
			name: "configmap-multiple-actions",
			fnconfig: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: policy
data:
  policy:
  - allow:
      and:
      - email:
          is: user@domain.com
  - deny:
      or:
      - groups:
          has: blocked
`,
			expectedActionCount: 2,
		},
		{
			name: "pomerium-policy-no-data",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
`,
		},
		{
			name: "pomerium-policy-empty-policy",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy: []
`,
		},
		{
			name: "pomerium-policy-single-action",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			expectedActionCount: 1,
		},
		{
			name: "pomerium-policy-multiple-actions",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
- deny:
    or:
    - groups:
        has: blocked
`,
			expectedActionCount: 2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var fnconfig *yaml.RNode
			if test.fnconfig == "" {
				fnconfig = nil
			} else {
				fnconfig = yaml.MustParse(test.fnconfig)
			}

			fn, err := New(fnconfig)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg, test.name)
			} else {
				assert.NoError(t, err, test.name)
				assert.NotNil(t, fn, test.name)
				assert.Equal(t, test.expectedActionCount, len(fn.Policy), test.name)
			}
		})
	}

}

type test struct {
	name           string
	fnconfig       string
	input          string
	expectedOutput string
	errorMsg       string
	resultCount    int
	resultSeverity framework.Severity
}

func TestPomeriumPolicy(t *testing.T) {
	var tests = []test{
		{
			name: "single-ingress",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
  annotations:
    ingress.pomerium.io/policy: '[{"allow":{"and":[{"email":{"is":"user@domain.com"}}]}}]'
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			resultCount:    1,
			resultSeverity: framework.Info,
		},
		{
			name: "multiple-ingresses",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: other-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app2.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app2
            port:
              number: 80
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
  annotations:
    ingress.pomerium.io/policy: '[{"allow":{"and":[{"email":{"is":"user@domain.com"}}]}}]'
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: other-ingress
  annotations:
    ingress.pomerium.io/policy: '[{"allow":{"and":[{"email":{"is":"user@domain.com"}}]}}]'
spec:
  ingressClassName: pomerium
  rules:
  - host: app2.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app2
            port:
              number: 80
`,
			resultCount:    2,
			resultSeverity: framework.Info,
		},
		{
			name: "no-ingress",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
`,
			resultCount:    1,
			resultSeverity: framework.Warning,
		},
		{
			name: "invalid-policy",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    andd:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			resultCount:    1,
			resultSeverity: framework.Error,
		},
		{
			name: "ingress-extensions-v1beta1",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: app-ingress
  annotations:
    ingress.pomerium.io/policy: '[{"allow":{"and":[{"email":{"is":"user@domain.com"}}]}}]'
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			resultCount:    1,
			resultSeverity: framework.Info,
		},
		{
			name: "ingress-networking-v1beta1",
			fnconfig: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
policy:
- allow:
    and:
    - email:
        is: user@domain.com
`,
			input: `
---
apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			expectedOutput: `apiVersion: v1
kind: Service
metadata:
  name: aservice
spec:
  ports:
  - port: 8080
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: app-ingress
  annotations:
    ingress.pomerium.io/policy: '[{"allow":{"and":[{"email":{"is":"user@domain.com"}}]}}]'
spec:
  ingressClassName: pomerium
  rules:
  - host: app.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              number: 80
`,
			resultCount:    1,
			resultSeverity: framework.Info,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err, "kio.ParseAll") {
				t.FailNow()
			}

			fn, err := New(yaml.MustParse(test.fnconfig))
			if !assert.NoError(t, err, "New") {
				t.FailNow()
			}

			outnodes, err := fn.Filter(input)
			if !assert.NoError(t, err, "Filter") {
				t.FailNow()
			}

			output, err := kio.StringAll(outnodes)
			if !assert.NoError(t, err, "kio.StringAll") {
				t.FailNow()
			}

			assert.Equal(t, test.expectedOutput, output, "unexpected output")

			results, err := fn.Results()
			if !assert.NoError(t, err, "Results") {
				t.FailNow()
			}
			assert.Equal(t, test.resultCount, len(results), "result count")
		})
	}
}
