---
title: "`kwok` in Cluster"
aliases:
  - /docs/user/kwok-in-cluster-old
---

# Deploy `kwok` in a Cluster

{{< hint "info" >}}

This document walks you through how to deploy `kwok` in a Kubernetes cluster.

{{< /hint >}}

{{< tabs "install-in-cluster" >}}

{{< tab "YAML" >}}

## Variables preparation

``` bash
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Deploy kwok and set up custom resource definitions (CRDs)

``` bash
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok.yaml"
```

## Set up default custom resources (CRs) of stages (required)

{{< hint "warning" >}}
NOTE: This configures the pod/node emulation behavior, if not it will do nothing.
{{< /hint >}}

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/stage-fast.yaml"
```

## Set up default custom resources (CRs) of resource usage (optional)

This allows to simulate the resource usage of nodes, pods and containers.

``` bash 
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/metrics-usage.yaml"
```

The above configuration sets the CPU and memory usage of all the containers managed by `kwok` to `1m` and to `1Mi` respectively.
To override the defaults, you can add annotation `"kwok.x-k8s.io/usage-cpu"` (for cpu usage) and
`"kwok.x-k8s.io/usage-memory"` (for memory usage) with any quantity value you want to the fake pods.

The resource usage simulation used above is annotation-based and the configuration is available at [here][resource usage from annotation].
For the explanation of how it works and more complex resource usage simulation methods, please refer to [ResourceUsage configuration].

{{< /tab >}}

{{< tab "Helm Chart (WIP)" >}}

The kwok helm chart is listed on the [artifact hub](https://artifacthub.io/packages/helm/kwok/kwok).

{{< /tab >}}

{{< tab "Kustomize (<0.4)" >}}

## Variables preparation

``` bash
# Temporary directory
KWOK_WORK_DIR=$(mktemp -d)
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Render kustomization yaml

Firstly, generate a kustomization template yaml to the previously-defined temporary directory.

``` bash
cat <<EOF > "${KWOK_WORK_DIR}/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: registry.k8s.io/kwok/kwok
  newTag: "${KWOK_LATEST_RELEASE}"
resources:
- "https://github.com/${KWOK_REPO}/kustomize/kwok?ref=${KWOK_LATEST_RELEASE}"
EOF
```

Next, render it with the prepared variables.

``` bash
kubectl kustomize "${KWOK_WORK_DIR}" > "${KWOK_WORK_DIR}/kwok.yaml"
```

## `kwok` deployment

Finally, we're able to deploy `kwok`:

``` bash
kubectl apply -f "${KWOK_WORK_DIR}/kwok.yaml"
```

{{< /tab >}}

{{< /tabs>}}

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[resource usage from annotation]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/metrics/usage
[ResourceUsage configuration]: {{< relref "/docs/user/resource-usage-configuration" >}}
