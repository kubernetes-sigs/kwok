---
title: "`kwok` in Cluster"
---

# Deploy `kwok` in a Cluster

{{< hint "info" >}}

This document walks you through how to deploy `kwok` in a Kubernetes cluster.

{{< /hint >}}

## Variables preparation

``` bash
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Deployment kwok and set up CRDs

``` bash
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok.yaml"
```

## Set up default CRs of Stages (required)

{{< hint "warning" >}}
NOTE: This configures the pod/node emulation behavior, if not it will do nothing.
{{< /hint >}}

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/stage-fast.yaml"
```

## Old way to deploy kwok

Old way to deploy kwok is [here][kwok in cluster old].

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[kwok in cluster old]: {{< relref "/docs/user/kwok-in-cluster-old" >}}
