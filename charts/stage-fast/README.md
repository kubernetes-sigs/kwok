# Default stage policy of KWOK (Kubernetes WithOut Kubelet)

[KWOK](https://github.com/kubernetes-sigs/kwok/) - Simulates thousands of Nodes and Clusters.

## Installing the Chart

Before you can install the chart you will need to add the `kwok` repo to [Helm](https://helm.sh/).

```shell
helm repo add kwok https://kwok.sigs.k8s.io/charts/
```

After you've installed the repo you can install the chart.

```shell
helm upgrade --namespace kube-system --install kwok kwok/kwok
```

Set up default stage policy (required)
> NOTE: This configures the pod/node emulation behavior, if not it will do nothing.

```shell
helm upgrade --install kwok kwok/stage-fast
```
