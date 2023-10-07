---
title: Github Action
---

#  Github Action

{{< hint "info" >}}

This document walks through the steps to use in the Github Action.

{{< /hint >}}

## Usage

``` yaml
- uses: kubernetes-sigs/kwok@main
  with:
    # Required: The command to install ('kwok' or 'kwokctl')
    command: ''
    # Optional: Specific version of command to install, defaults to latest release
    kwok-version: ''
```

### Create Cluster

``` yaml
- name: Set up kwokctl
  uses: kubernetes-sigs/kwok@main
  with:
    command: kwokctl
- run: kwokctl create cluster
  env:
    KWOK_KUBE_VERSION: v1.28.0
```

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
