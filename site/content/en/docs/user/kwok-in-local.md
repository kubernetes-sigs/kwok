# Run Kwok in the Local

{{< hint "info" >}}

This document walks you through how to run `kwok` in the local.

{{< /hint >}}

## Install Kwok

[Install Kwok]({{< relref "/docs/user/install" >}}) in the local.

## Run Kwok in the local

Finally, we're able to run `kwok` in the local for a cluster and maintain their heartbeats:

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
