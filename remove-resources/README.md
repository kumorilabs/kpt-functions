# remove-resources

## Overview

## Usage

The `remove-resources` function removes the input resources. It relies on [Kpt][Kpt] [`selectors`][selectors] to only invoke the function on the subset of resources you want to remove. If you don't specify `selectors`, the `remove-resources` function will happily remove all resources.


Example:

``` yaml
pipeline:
  mutators:
    - image: ghcr.io/kumorilabs/krm-fn-remove-resources:0.1
      selectors:
        - kind: Secret
```

The above pipeline will remove all `Secret` resources from the package.

[Kpt]:https://kpt.dev/

[selectors]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution?id=specifying-selectors
