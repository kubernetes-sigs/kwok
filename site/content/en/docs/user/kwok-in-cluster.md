---
title: "`kwok` in Cluster"
aliases:
- /docs/user/kwok-in-cluster-old
---

# Deploy `kwok` in a Cluster

{{< hint "info" >}}

This document walks you through how to deploy `kwok` in a Kubernetes cluster.

{{< /hint >}}


## Helm Chart Installation

The kwok helm chart is listed on the [artifact hub](https://artifacthub.io/packages/helm/kwok/kwok).

## YAML Installation

You can also install `kwok` using the provided YAML manifests.

### Variables preparation

``` bash
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

### Deploy kwok and set up custom resource definitions (CRDs)

``` bash
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok.yaml"
```

### Set up default custom resources (CRs) of stages (required)

{{< hint "warning" >}}

NOTE: This step is required to configure the pod/node simulation behavior.

{{< /hint >}}

The default simulation behavior is defined in [fast stage for pod][fast-stage-pod] and [fast stage for node][fast-stage-node].
They are suitable for scenarios such as scheduling and autoscaling.
You can customize the simulation behavior with your own configuration file.

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/stage-fast.yaml"
```

### Set up default custom resources (CRs) of resource usage (optional)

This allows to simulate the resource usage of nodes, pods and containers.

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/metrics-usage.yaml"
```

The above configuration sets the CPU and memory usage of all the containers managed by `kwok` to `1m` and to `1Mi` respectively.
To override the defaults, you can add annotation `"kwok.x-k8s.io/usage-cpu"` (for cpu usage) and
`"kwok.x-k8s.io/usage-memory"` (for memory usage) with any quantity value you want to the fake pods.

The resource usage simulation used above is annotation-based and the configuration is available at [here]([metrics-usage]).
For the explanation of how it works and more complex resource usage simulation methods, please refer to [ResourceUsage configuration].

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[fast-stage-pod]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/pod/fast
[fast-stage-node]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/node/fast
[metrics-usage]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/metrics/usage
[ResourceUsage configuration]: {{< relref "/docs/user/resource-usage-configuration" >}}
