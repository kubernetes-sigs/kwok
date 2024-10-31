---
title: ResourceUsage
---

# ResourceUsage Configuration

{{< hint "info" >}}

This document walks you through how to simulate the resource usage of pod(s).

{{< /hint >}}

## What is a ResourceUsage?

The [ResourceUsage] is a [`kwok` Configuration][configuration] that allows users to define and simulate the resource usages of a single pod.

The YAML below shows all the fields of a ResourceUsage resource:

``` yaml
kind: ResourceUsage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
  namespace: <string>
spec:
  usages:
  - containers:
    - <string>
    usage:
      cpu:
        value: <quantity>
        expression: <string>
      memory:
        value: <quantity>
        expression: <string>
```

To associate a ResourceUsage with a certain pod to be simulated, users must ensure `metadata.name` and `metadata.namespace` 
are inconsistent with the name and namespace of the target pod.

The resource usages of a pod are specified via `usages` field.
The `usages` field are organized by groups, with each corresponding to a collection of containers that shares a same resource usage simulation setting.
Each group consists of a list of container names (`containers`) and the shared resource usage setting (`usage`).

{{< hint "info" >}}
If `containers` is not given in a group, the `usage` in that group will be applied to all containers of the target pod.
{{< /hint >}}

You can simply set a static [Quantity value] (`100Mi`, `1000m`, etc.) via `cpu.value` and `memory.value` to define the cpu and memory resource usage respectively.
Besides, users are also allowed to provide a [CEL expression] via `expressions` to describe the resource usage more flexibly.
For example, the following expression tries to extract the cpu resource usage from the pod's annotation if it has or use a default value.

```yaml
expression: |
  "kwok.x-k8s.io/usage-cpu" in pod.metadata.annotations
  ? Quantity(pod.metadata.annotations["kwok.x-k8s.io/usage-cpu"])
  : Quantity("1m")
```

{{< hint "info" >}}
1. `value` has higher priority than `expressions` if both are set.
2. Quantity value must be explicitly wrapped by `Quantity` function in CEL expressions.
{{< /hint >}}

With CEL expressions, it is even possible to simulate resource usages dynamically. For example, the following expression
yields memory usage that grows linearly with time.
```yaml
expression: 'Quantity("1Mi") * (pod.SinceSecond() / 60.0)'
```
Please refer to [CEL expressions in `kwok`][CEL expressions] for an exhausted list that may be helpful to configure dynamic resource usage.

### ClusterResourceUsage

In addition to simulating a single pod, users can also simulate the resource usage for multiple pods via [ClusterResourceUsage].

The YAML below shows all the fields of a ClusterResourceUsage resource:

``` yaml
kind: ClusterResourceUsage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  selector:
    matchNamespaces:
    - <string>
    matchNames:
    - <string>
  usages:
  - containers:
    - <string>
    usage:
    cpu:
      value: <quantity>
      expression: <string>
    memory:
      value: <quantity>
      expression: <string>
```
Compared to ResourceUsage, whose `metadata.name` and `metadata.namespace` are required to match the associated pod, 
ClusterResourceUsage has an additional `selector` field for specifying the target pods to be simulated.
`matchNamespaces` and `matchNames` are both represented as list，which are designed to take pod collections by different levels:

1. If `matchNamespaces` is empty, ClusterResourceUsage will be applied to all pods that are managed by `kwok` and whose names listed in `matchNames`.
2. If `matchNames` is empty, ClusterResourceUsage will be applied to all pods managed by `kwok` and under namespaces listed in `matchNamespaces`.
3. If `matchNames` and `matchNamespaces` are both unset, ClusterResourceUsage will be applied to all pods that `kwok` manages.

The `usages` field of ClusterResourceUsage has the same semantic with the one in ResourceUsage.

## Dependencies

- [Metrics] and [`/metrics/resource` endpoint][metrics resource endpoint]
- metrics-server version >= 0.7.0
  it will get the annotation `metrics.k8s.io/resource-metrics-path` for the node instead of the default `/metrics/resource` endpoint.

### Where to get the simulation data?

The data can be fetched from the metric service of `kwok` with the path `/metrics/nodes/{nodeName}/metrics/resource`,
where `{nodeName}` is the name of the fake node.
The metrics returned are similar to the response from kubelet's `/metrics/resource` endpoint.

Please refer to [`kwok` Metrics][Metrics] about how to integrate `kwok` simulated metrics endpoints with metrics-server.  

## Out-of-the-box

Currently, a configuration is provided to quickly simulate the resource usage of pods.

- [Metrics resource endpoint][metrics resource endpoint]
- [Resource usage from annotation][resource usage from annotation]

```yaml
# the <nodeName> should be replaced with the name of the node
kind: Node
apiVersion: v1
metadata:
  annotations:
    metrics.k8s.io/resource-metrics-path: "/metrics/nodes/<nodeName>/metrics/resource"
...
```

```yaml
# specify the resource usage of a pod via annotation
kind: Pod
apiVersion: v1
metadata:
  annotations: 
    kwok.x-k8s.io/usage-cpu: "100m"
    kwok.x-k8s.io/usage-memory: "100Mi"
...
```

<img width="700px" src="/img/demo/resource-usage.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[ResourceUsage]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ResourceUsage
[ClusterResourceUsage]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage
[Quantity value]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes
[CEL expression]: https://github.com/google/cel-spec/blob/master/doc/langdef.md
[Metrics]: {{< relref "/docs/user/metrics-configuration" >}}
[metrics resource endpoint]: https://github.com/kubernetes-sigs/kwok/blob/main/kustomize/metrics/resource/metrics-resource.yaml
[resource usage from annotation]: https://github.com/kubernetes-sigs/kwok/blob/main/kustomize/metrics/usage/usage-from-annotation.yaml
[CEL expressions]: {{< relref "/docs/user/cel-expressions" >}}
