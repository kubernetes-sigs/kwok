---
title: "`kwok` out of Cluster"
aliases:
- /docs/user/kwok-in-local/
---

# Running `kwok` out of cluster

{{< hint "info" >}}

This document walks you through how to run `kwok` out of cluster for a Kubernetes cluster.

{{< /hint >}}

## Prerequisites

Firstly, you need to have a Kubernetes cluster, and the `kwok` command-line tool must be available in your `PATH`.

## Install `kwok`

[Install `kwok`][install] in your environment.

## Running

Next, you can run `kwok` out of cluster by using the `--kubeconfig` flag.

The default simulation behavior is defined in [fast stage for pod][fast-stage-pod] and [fast stage for node][fast-stage-node].
They are suitable for scenarios such as scheduling and autoscaling.
You can customize the simulation behavior with `--config` flag to point to your own configuration file.

```bash
kwok \
  --kubeconfig=~/.kube/config \
  --manage-all-nodes=false \
  --manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake \
  --manage-nodes-with-label-selector= \
  --manage-single-node= \
  --cidr=10.0.0.1/24 \
  --node-ip=10.0.0.1 \
  --node-lease-duration-seconds=40
```

Finally, you can see the `kwok` is running out of cluster for the Kubernetes cluster.

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[install]: {{< relref "/docs/user/installation" >}}
[fast-stage-pod]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/pod/fast
[fast-stage-node]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/node/fast
