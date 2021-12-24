# configmap-injector

## Overview

## Usage

The `configmap-injector` function includes two client-side custom resources that
let you inject keys into an existing `ConfigMap` (or to generate a new
`ConfigMap`). We often need a way to customize `ConfigMap` data for an instance
of our package. Since [KRM functions][KRM] can only operate on KRM-style
resources, it can be difficult to use the same functions to accomplish this. The
`configmap-injector` function lets us embed our configuration into client-side
KRM resources so that existing KRM functions (like `apply-setters`) can operate
against them.

### ConfigMapInject

Use `ConfigMapInject` when you have YAML-based configuration that needs to be
configurable and included in a `ConfigMap`. This resource lets you include the
strongly-typed YAML into a KRM resource. This means that you can use other KRM
functions (like `apply-setters`) to mutate the configuration. The
`configmap-injector` function will then inject the configuration into a new or
existing `ConfigMap`. The `metadata.name` (and `metadata.namespace`, if
included) identifies the target `ConfigMap`. All keys in the `data` map will get
injected into the target `ConfigMap`'s data field. If a key with the same name
already exists in the target `ConfigMap`, it will be overridden. If the target
`ConfigMap` doesn't exist, the function will generate it.

Example:

``` yaml
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
```

When the `configmap-injector` function runs, it will inject the contents of
`repository.credentials` into the "argocd-cm" `ConfigMap` as a string. Your
`Kptfile` pipeline might look something like this:

``` yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2
      configMap:
        git-base-url: https://github.com/kumorilabs
    - image: ghcr.io/kumorilabs/krm-fn-configmap-injector:0.3
```

After invoking the pipeline (`kpt fn render`), the "argocd-cm" `ConfigMap` would
end up looking like this:

``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
data:
  admin.enabled: "false"
  repository.credentials: |
    - passwordSecret:
        key: password
        name: git-reader
      url: https://github.com/kumorilabs
      usernameSecret:
        key: username
        name: git-reader
  ... additional fields here ...
```

### ConfigMapTemplate

Use `ConfigMapTemplate` when you have non-YAML configuration that you need to
customize and include in a `ConfigMap`. Because KRM functions only operate on
YAML, this resource let's us define our non-YAML configuration as a template
embedded in a KRM resource. We can then use other KRM functions (like
`apply-setters`) to configure the values used to render the template.

Like `ConfigMapInject`, the `metadata.name` (and `metadata.namespace`, if
included) identifies the target `ConfigMap`. The function will render all
templates in the `data` map using the values defined in the `values` map. The
rendered template(s) will get injected into the target `ConfigMap`'s data field.
If a key with the same name already exists in the target `ConfigMap`, it will be
overridden. If the target `ConfigMap` doesn't exist, the function will generate
it.

Example:

``` yaml
apiVersion: fn.kumorilabs.io/v1alpha1
kind: ConfigMapTemplate
metadata:
  name: some-cm
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  config.json: |
    {
      "id": "v1",
      "log-level": "{{.logLevel}}",
      "base-url": "{{.baseUrl}}",
    }
values:
  logLevel: debug # kpt-set: ${log-level}
  baseUrl: https://github.com/kumorilabs # kpt-set: ${base-url}
```

When the `configmap-injector` function runs, it will inject the rendered
template defined in `config.json` into the "some-cm" `ConfigMap` as a string.
Your `Kptfile` pipeline might look something like this:

``` yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2
      configMap:
        log-level: debug
        base-url: https://github.com/kumorilabs
    - image: ghcr.io/kumorilabs/krm-fn-configmap-injector:0.3
```

After invoking the pipeline (`kpt fn render`), the "some-cm" `ConfigMap` would
end up looking something like:

``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-cm
data:
  config.json: |
    {
      "id": "v1",
      "log-level": "debug",
      "base-url": "https://github.com/kumorilabs",
    }
```

## Notes

* You can use multiple `ConfigMapInject` or `ConfigMapTemplate` resources and
  mix and match them as needed in your package, even if they are targeting the
  same `ConfigMap`. Of course, if you specify the same keys in multiple
  resources that target the same `ConfigMap`, the last one that runs will "win".
* `ConfigMapInject` resources get processed before `ConfigMapTemplate`
  resources.
* Use the `config.kubernetes.io/local-config: "true"` annotation on the
  `ConfigMapInject` and `ConfigMapTemplate` resources to signal to other tools
  (like [Kustomize][Kustomize]) to exclude the resource from their output. If
  you try to apply one of these resources to the Kubernetes API server, you will
  get an error since these are not server-side custom resources.


[KRM]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md

[Kustomize]: https://kustomize.io/
