---
title: ResourceUsage
---

# ResourceUsage Configuration

{{< hint "info" >}}

This document walks you through how to configure the ResourceUsage feature.

{{< /hint >}}

## What is ResourceUsage?

The [ResourceUsage API] is a [`kwok` Configuration][configuration] that allows users to define and simulate the resource usage of Pod(s).

A ResourceUsage resource has the following fields:

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
        expression: <string>
      memory:
        expression: <string>
```

To simulate the resource usage, you can set the `usages` field in the spec section of a ResourceUsage resource.
The `containers` field is used to match an item in the `usages` field. If the `containers` field is not set, the `usages` item will default to all containers.
The `usage` field specifies the resource usage to be simulated. The `cpu` field is used to simulate the CPU usage, and the `memory` field is used to simulate the memory usage.


### ClusterResourceUsage

The [ClusterResourceUsage API] is a special ResourceUsage API which is cluster-side.

A ClusterResourceUsage resource has the following fields:

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
      expression: <string>
    memory:
      expression: <string>
```

The `selector` field is used to select the Pods to simulate the resource usage.
The `matchNamespaces` field is used to match the namespace of the Pods. If the `matchNamespaces` field is not set, the ClusterResourceUsage will match all namespaces.
The `matchNames` field is used to match the name of the Pods. If the `matchNames` field is not set, the ClusterResourceUsage will match all Pods.

## Dependencies

[Metrics] is required to be enabled and set up default CRs for ResourceUsage

[configuration]: {{< relref "/docs/user/configuration" >}}
[ResourceUsage API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ResourceUsage
[ClusterResourceUsage API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage
[Metrics]: {{< relref "/docs/user/metrics-configuration" >}}
