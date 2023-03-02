# Run `kwok` in Your Local

{{< hint "info" >}}

This document walks you through how to run `kwok` in your local.

{{< /hint >}}

After you have [`kwok` installed]({{< relref "/docs/user/install" >}}) in a local computer,
you can run `kwok` immediately for clusters and maintain their heartbeats:

``` bash
kwok \
  --kubeconfig=~/.kube/config \
  --manage-all-nodes=false \
  --manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake \
  --manage-nodes-with-label-selector= \
  --disregard-status-with-annotation-selector=kwok.x-k8s.io/status=custom \
  --disregard-status-with-label-selector= \
  --cidr=10.0.0.1/24 \
  --node-ip=10.0.0.1
```
