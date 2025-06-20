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

Set up default mutating webhook to manage pods with selector (optional)

```shell
CA_BUNDLE=$(kubectl get secret -n kube-system kwok-controller -o jsonpath='{.data.ca\.crt}')
kubectl patch mutatingwebhookconfiguration kwok-controller \
  --type='json' \
  -p="[{'op': 'replace', 'path': '/webhooks/0/clientConfig/caBundle', 'value':'${CA_BUNDLE}'}]"
```

## Configuration

The following table lists the configurable parameters of the kwok chart and their default values.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| enableDeployment | bool | `true` |  |
| env[0].name | string | `"POD_IP"` |  |
| env[0].valueFrom.fieldRef.fieldPath | string | `"status.podIP"` |  |
| env[1].name | string | `"HOST_IP"` |  |
| env[1].valueFrom.fieldRef.fieldPath | string | `"status.hostIP"` |  |
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
| tolerations[0].effect | string | `"NoSchedule"` |  |
| tolerations[0].key | string | `"node-role.kubernetes.io/control-plane"` |  |
| tolerations[0].operator | string | `"Exists"` |  |
| tolerations[1].effect | string | `"NoSchedule"` |  |
| tolerations[1].key | string | `"node-role.kubernetes.io/master"` |  |
| tolerations[1].operator | string | `"Exists"` |  |
| volumeMounts | list | `[]` |  |
| volumes | list | `[]` |  |
| server.port | int | 10247 |  |
| server.enableTLS | bool | `true` |  |
| server.managePodsWithSelector.matchLabels | object | `{app: fake-pod}` |  |
| server.managePodsWithSelector.excludedNamespaces | list | `{}` |  |
