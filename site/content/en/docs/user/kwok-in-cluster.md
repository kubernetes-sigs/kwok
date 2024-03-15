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

## Set up default CRs of resource usage (optional)

This allows to simulate the resource usage of nodes, pods and containers.

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/metrics-usage.yaml"
```

The above configuration sets the CPU and memory usage of all the containers managed by `kwok` to `1m` and to `1Mi` respectively.
To override the defaults, you can add annotation `"kwok.x-k8s.io/usage-cpu"` (for cpu usage) and
`"kwok.x-k8s.io/usage-memory"` (for memory usage) with any quantity value you want to the fake pods.

The resource usage simulation used above is annotation-based and the configuration is available at [here][resource usage from annotation].
For the explanation of how it works and more complex resource usage simulation methods, please refer to [ResourceUsage configuration].

## Old way to deploy kwok

Old way to deploy kwok is [here][kwok in cluster old].

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[kwok in cluster old]: {{< relref "/docs/user/kwok-in-cluster-old" >}}
[resource usage from annotation]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/metrics/usage
[ResourceUsage configuration]: {{< relref "/docs/user/resource-usage-configuration" >}}
