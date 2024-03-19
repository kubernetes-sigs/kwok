---
title: Attach
---

# Attach Configuration

{{< hint "info" >}}

This document walks you through how to configure the Attach feature.

{{< /hint >}}

## What is an Attach?

The [Attach] is a [`kwok` Configuration][configuration] that allows users to define and simulate attaching to a single pod.

The YAML below shows all the fields of an Attach resource:

``` yaml
kind: Attach
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
  namespace: <string>
spec:
  attaches:
  - containers:
    - <string>
    logsFile: <string>
```

To associate an Attach with a certain pod to be simulated, users must ensure `metadata.name` and `metadata.namespace`
are inconsistent with the name and namespace of the target pod.

The attaching simulation setting of a pod are specified via `attaches` field.
The `attaches` field is organized by groups, with each corresponding to a collection of containers that shares a same attaching simulation setting.
Each group consists of a list of container names (`containers`) and the shared attaching simulation setting (`logsFile`).

{{< hint "info" >}}
If `containers` is not given in a group, the `logsFile` in that group will be applied to all containers of the target pod.
{{< /hint >}}

The `logsFile` field specifies the file path of the logs. If the `logsFile` field is not set, this item will be ignored.

### ClusterAttach

In addition to simulating a single pod, users can also simulate the attaching for multiple pods via [ClusterAttach].

The YAML below shows all the fields of a ClusterAttach resource:

``` yaml
kind: ClusterAttach
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  selector:
    matchNamespaces:
    - <string>
    matchNames:
    - <string>
  attaches:
  - containers:
    - <string>
    logsFile: <string>
```

Compared to Attach, whose `metadata.name` and `metadata.namespace` are required to match the associated pod,
ClusterAttach has an additional `selector` field for specifying the target pods to be simulated.
`matchNamespaces` and `matchNames` are both represented as list，which are designed to take pod collections by different levels:

1. If `matchNamespaces` is empty, ClusterAttach will be applied to all pods that are managed by `kwok` and whose names listed in `matchNames`.
2. If `matchNames` is empty, ClusterAttach will be applied to all pods managed by `kwok` and under namespaces listed in `matchNamespaces`.
3. If `matchNames` and `matchNamespaces` are both unset, ClusterAttach will be applied to all pods that `kwok` manages.

The `attaches` field of ClusterAttach has the same semantic with the one in Attach.

## Examples

<img width="700px" src="/img/demo/attach.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[Attach]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Attach
[ClusterAttach]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterAttach
