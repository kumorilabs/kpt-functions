package filters

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
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
			name:        "single package",
			resultCount: 1,
			context:     "test/upstream",
			input: `
apiVersion: fn.kumorilabs.io/v1alpha1
kind: KRMPackage
metadata:
  name: test-package
  annotations:
	config.kubernetes.io/local-config: "true"
spec:
  artifacts:
  - package: synax.azurecr.io/test-package/test:0.1.0
    context: test/upstream
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
	app.kubernetes.io/component: redis
	app.kubernetes.io/name: argocd-redis
	app.kubernetes.io/part-of: argocd
  name: argocd-redis
  namespace: kumori-argocd
spec:
  selector:
	matchLabels:
	  app.kubernetes.io/name: argocd-redis
  template:
	metadata:
	  labels:
		app.kubernetes.io/name: argocd-redis
	spec:
	  affinity:
		podAntiAffinity:
		  preferredDuringSchedulingIgnoredDuringExecution:
		  - podAffinityTerm:
			  labelSelector:
				matchLabels:
				  app.kubernetes.io/name: argocd-redis
			  topologyKey: kubernetes.io/hostname
			weight: 100
		  - podAffinityTerm:
			  labelSelector:
				matchLabels:
				  app.kubernetes.io/part-of: argocd
			  topologyKey: kubernetes.io/hostname
			weight: 5
	  containers:
	  - args:
		- --save
		- ""
		- --appendonly
		- "no"
		image: redis:6.2.6-alpine
		imagePullPolicy: Always
		name: redis
		ports:
		- containerPort: 6379
	  securityContext:
		runAsNonRoot: true
		runAsUser: 999
	  serviceAccountName: argocd-redis
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
	app.kubernetes.io/component: application-controller
	app.kubernetes.io/name: argocd-application-controller
	app.kubernetes.io/part-of: argocd
  name: argocd-application-controller
  namespace: kumori-argocd
spec:
  replicas: 1
  selector:
	matchLabels:
	  app.kubernetes.io/name: argocd-application-controller
  serviceName: argocd-application-controller
  template:
	metadata:
	  labels:
		app.kubernetes.io/name: argocd-application-controller
	spec:
	  affinity:
		podAntiAffinity:
		  preferredDuringSchedulingIgnoredDuringExecution:
		  - podAffinityTerm:
			  labelSelector:
				matchLabels:
				  app.kubernetes.io/name: argocd-application-controller
			  topologyKey: kubernetes.io/hostname
			weight: 100
		  - podAffinityTerm:
			  labelSelector:
				matchLabels:
				  app.kubernetes.io/part-of: argocd
			  topologyKey: kubernetes.io/hostname
			weight: 5
	  containers:
	  - command:
		- argocd-application-controller
		env:
		- name: ARGOCD_RECONCILIATION_TIMEOUT
		  valueFrom:
			configMapKeyRef:
			  key: timeout.reconciliation
			  name: argocd-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER
		  valueFrom:
			configMapKeyRef:
			  key: repo.server
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER_TIMEOUT_SECONDS
		  valueFrom:
			configMapKeyRef:
			  key: controller.repo.server.timeout.seconds
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_STATUS_PROCESSORS
		  valueFrom:
			configMapKeyRef:
			  key: controller.status.processors
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_OPERATION_PROCESSORS
		  valueFrom:
			configMapKeyRef:
			  key: controller.operation.processors
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_LOGFORMAT
		  valueFrom:
			configMapKeyRef:
			  key: controller.log.format
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_LOGLEVEL
		  valueFrom:
			configMapKeyRef:
			  key: controller.log.level
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_METRICS_CACHE_EXPIRATION
		  valueFrom:
			configMapKeyRef:
			  key: controller.metrics.cache.expiration
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_SELF_HEAL_TIMEOUT_SECONDS
		  valueFrom:
			configMapKeyRef:
			  key: controller.self.heal.timeout.seconds
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER_PLAINTEXT
		  valueFrom:
			configMapKeyRef:
			  key: controller.repo.server.plaintext
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER_STRICT_TLS
		  valueFrom:
			configMapKeyRef:
			  key: controller.repo.server.strict.tls
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_APP_STATE_CACHE_EXPIRATION
		  valueFrom:
			configMapKeyRef:
			  key: controller.app.state.cache.expiration
			  name: argocd-cmd-params-cm
			  optional: true
		- name: REDIS_SERVER
		  valueFrom:
			configMapKeyRef:
			  key: redis.server
			  name: argocd-cmd-params-cm
			  optional: true
		- name: REDISDB
		  valueFrom:
			configMapKeyRef:
			  key: redis.db
			  name: argocd-cmd-params-cm
			  optional: true
		- name: ARGOCD_DEFAULT_CACHE_EXPIRATION
		  valueFrom:
			configMapKeyRef:
			  key: controller.default.cache.expiration
			  name: argocd-cmd-params-cm
			  optional: true
		image: quay.io/argoproj/argocd:v2.3.0-rc5
		imagePullPolicy: Always
		livenessProbe:
		  httpGet:
			path: /healthz
			port: 8082
		  initialDelaySeconds: 5
		  periodSeconds: 10
		name: argocd-application-controller
		ports:
		- containerPort: 8082
		readinessProbe:
		  httpGet:
			path: /healthz
			port: 8082
		  initialDelaySeconds: 5
		  periodSeconds: 10
		securityContext:
		  allowPrivilegeEscalation: false
		  capabilities:
			drop:
			- all
		  readOnlyRootFilesystem: true
		  runAsNonRoot: true
		volumeMounts:
		- mountPath: /app/config/controller/tls
		  name: argocd-repo-server-tls
		- mountPath: /home/argocd
		  name: argocd-home
		workingDir: /home/argocd
	  serviceAccountName: argocd-application-controller
	  volumes:
	  - emptyDir: {}
		name: argocd-home
	  - name: argocd-repo-server-tls
		secret:
		  items:
		  - key: tls.crt
			path: tls.crt
		  - key: tls.key
			path: tls.key
		  - key: ca.crt
			path: ca.crt
		  optional: true
		  secretName: argocd-repo-server-tls
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
	app.kubernetes.io/name: argocd-cm
	app.kubernetes.io/part-of: argocd
  name: argocd-cm
  namespace: kumori-argocd
`,

			expected: `
`,
		},
	}
	runTests(t, tests)
}

func runTests(t *testing.T, tests []test) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := os.MkdirAll(test.context, os.ModePerm); err != nil {
				t.FailNow()
			}
			baseDir, err := ioutil.TempDir(test.context, "")
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			test.expected = test.input

			r, err := ioutil.TempFile(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			defer os.Remove(r.Name())
			err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			filter := &KRMPackageFilter{}
			inout := &kio.LocalPackageReadWriter{
				PackagePath: baseDir,
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

		})
	}
}
