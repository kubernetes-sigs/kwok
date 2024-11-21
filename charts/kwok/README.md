# KWOK (Kubernetes WithOut Kubelet)

[KWOK](https://kwok.sigs.k8s.io/) - Simulates thousands of Nodes and Clusters.

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

Set up default metrics usage policy (optional)

```shell
helm upgrade --install kwok kwok/metrics-usage
```

## Configuration

The following table lists the configurable parameters of the kwok chart and their default values.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| fullnameOverride | string | `"kwok-controller"` | Override the `fullname` of the chart. |
| hostNetwork | bool | `false` | Change `hostNetwork` to `true` if you want to deploy in a kind cluster. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.repository | string | `"registry.k8s.io/kwok/kwok"` | Image repository. |
| image.tag | string | `""` | Overrides the image tag whose default is {{ .Chart.AppVersion }}. |
| imagePullSecrets | list | `[]` | Image pull secrets. |
| nameOverride | string | `""` | Override the `name` of the chart. |
| nodeSelector | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| replicas | int | `1` | The replica count for Deployment. |
| resources | object | `{}` |  |
| securityContext | object | `{}` |  |
