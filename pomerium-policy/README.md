# pomerium-policy

## Overview

The `pomerium-policy` function validates and injects [Pomerium][Pomerium]
[policies][policies] into `Ingress` resources. When using the [Pomerium Ingress
Controller][PomeriumIngressController], you define a policy by writing it (in
JSON or YAML) into a `ingress.pomerium.io/policy` annotation on the `Ingress`
resource. This can be error-prone and challenging to author. This function
allows you to author a Pomerium policy in plain YAML in a custom, client-side
[KRM][KRM] resource. This unlocks the ability to use other KRM functions (like
setters) against your policies. The `pomerium-policy` function will parse your
policy and, if valid, inject it (serialized to JSON) into your `Ingress`
resource(s).

## Usage

policy.yaml:

``` yaml
apiVersion: fn.kumorilabs.io/v1alpha1
kind: PomeriumPolicy
metadata:
  name: policy
  annotations:
    config.kubernetes.io/local-config: "true"
policy:
- allow:
    and:
    - email:
        is: user@domain.com
```

Function input:

``` yaml
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
```

[Kpt][Kpt] [pipeline][pipeline]:

``` yaml
pipeline:
  mutators:
    - image: ghcr.io/kumorilabs/krm-fn-pomerium-policy:0.1
      configPath: policy.yaml
```

When the `pomerium-policy` function runs, it will parse the contents of `policy`
as a Pomerium policy. If it is valid, it will inject that policy into a
`ingress.pomerium.io/policy` annotation on all `Ingress` resources in your
input. The resulting `Ingress` resource should look like this:

``` yaml
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
```

### Define Policy in a ConfigMap

This function also supports authoring your policy in a `ConfigMap` resource.

``` yaml
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
```

Note that you specify the policy in `.data.policy` in the `ConfigMap`. This also
means you can include a policy inline in your pipeline if you wish:

``` yaml
pipeline:
  mutators:
    - image: ghcr.io/kumorilabs/krm-fn-pomerium-policy:0.1
      configMap:
        policy:
        - allow:
            and:
            - email:
                is: user@domain.com
```

### Different Policies per Ingress

If you have multiple `Ingress` resources in your input and you want to apply
different policies to them, you can run multiple instances of the function and
use [selectors][selectors] to target the appropriate Ingress resource.

For example, your Kpt pipeline could look something like this:

``` yaml
pipeline:
  mutators:
    - image: ghcr.io/kumorilabs/krm-fn-pomerium-policy:0.1
      configPath: policy-users.yaml
      selectors:
        - kind: Ingress
          name: user-app
    - image: ghcr.io/kumorilabs/krm-fn-pomerium-policy:0.1
      configPath: policy-admins.yaml
      selectors:
        - kind: Ingress
          name: admin-app
```

[KRM]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md

[selectors]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution?id=specifying-selectors

[Kpt]: https://kpt.dev/

[pipeline]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution

[Pomerium]: https://www.pomerium.com/

[policies]: https://www.pomerium.com/docs/topics/ppl.html

[PomeriumIngressController]: https://www.pomerium.com/docs/k8s/ingress.html
