# Configuration

{{< hint "info" >}}

This document walks you through how to configure `kwok`/`kwokctl` options.

{{< /hint >}}

## Getting Started

To configure, you will need to create a [YAML](https://yaml.org/) configuration file.
This file follows Kubernetes conventions for versioning etc.

A minimal valid config file looks like this:

``` yaml
kind: KwokConfiguration
apiVersion: kwok.x-k8s.io/v1alpha1
options:
---
kind: KwokctlConfiguration
apiVersion: kwok.x-k8s.io/v1alpha1
options:
```

This config specifies that we are configuring `kwok`/`kwokctl` and that the version of the config we are using is `v1alpha1` (`apiVersion: kwok.x-k8s.io/v1alpha1`).

Different versions may support different options and behaviors, which is why we must always specify the version. This mechanism is inspired by Kubernetes resources and component config.

To use this config, place the contents in a file `~/.kwok/kwok.yaml` or run command with `--config=kwok.yaml` from the same directory.

The structure of the Configuration type is defined by a Go struct, which is described [here](https://pkg.go.dev/sigs.k8s.io/kwok/pkg/apis/v1alpha1).

## A Note on CLI Flags, Environment Variables, and Configuration Files

Uses the following precedence order. Each item takes precedence over the item below it:

1. flags specified on the command line
2. environment variables (with the prefix `KWOK_`)
3. values specified in the configuration file (`--config=` or `~/.kwok/kwok.yaml`)
4. default values

## Using `kwok`

When using `kwok`, it takes the configuration about itself from the configuration file and ignores all other configurations.

## Using `kwokctl`

When using `kwokctl`, it takes the configuration about itself from the configuration file and passes the configuration file to `kwok`.
