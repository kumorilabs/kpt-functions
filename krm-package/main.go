package main

import (
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
)

func main() {

	p := KRMPackageProcessor{}
	cmd := command.Build(&p, command.StandaloneEnabled, false)

	cmd.Short = "Pulls or Pushes OCI Native KRM Packages"
	cmd.Long = "Pulls or Pushes OCI Native KRM Packages"

	if err := cmd.Execute(); err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
