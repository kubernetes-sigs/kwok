# KWOK (Kubernetes WithOut Kubelet)

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

## Configuration

The following table lists the configurable parameters of the kwok chart and their default values.

| Parameter          | Description                                                                  | Default                     |
|--------------------|------------------------------------------------------------------------------|-----------------------------|
| `image.repository` | Image repository.                                                            | `registry.k8s.io/kwok/kwok` |
| `image.tag`        | Image tag, will override the default tag derived from the chart app version. | `[chart appVersion]`        |
| `image.pullPolicy` | Image pull policy.                                                           | `IfNotPresent`              |
| `imagePullSecrets` | Image pull secrets.                                                          | `[]`                        |
| `nameOverride`     | Override the `name` of the chart.                                            | `""`                        |
| `fullnameOverride` | Override the `fullname` of the chart.                                        | `""`                        |
| `replicas`         | The replica count for Deployment.                                            | `1`                         |
