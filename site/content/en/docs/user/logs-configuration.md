---
title: "Logs"
---

# Logs Configuration

{{< hint "info" >}}

This document walks you through how to configure the Logs feature.

{{< /hint >}}

## What is a Logs?

The [Logs] is a [`kwok` Configuration][configuration] that allows users to define and simulate logs to a single pod.

The YAML below shows all the fields of a Logs resource:

``` yaml
kind: Logs
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
  namespace: <string>
spec:
  logs:
  - containers:
    - <string>
    logsFile: <string>
    follow: <bool>
```
The logs simulation setting of a pod is specified via `logs` field.
The `logs` field is organized by groups, with each corresponding to a collection of containers that shares a same logs simulation config.
Each group consists of a list of container names (`containers`) and the shared simulation settings (`logsFile` and `follow`).

{{< hint "info" >}}
If `containers` is not given in a group, the `logsFile` and `follow` in that group will be applied to all containers of the target pod.
{{< /hint >}}

The `logsFile` field specifies the file path of the logs. If the `logsFile` field is not set, this item will be ignored.
The `follow` field specifies whether to follow the logs. If the `follow` field is not set, the `follow` field will default to false.

### ClusterLogs

In addition to simulating a single pod, users can also simulate the logs for multiple pods via [ClusterLogs].

The YAML below shows all the fields of a ClusterLogs resource:

``` yaml
kind: ClusterLogs
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  selector:
    matchNamespaces:
    - <string>
    matchNames:
    - <string>
  logs:
  - containers:
    - <string>
    logsFile: <string>
    follow: <bool>
```

Compared to Logs, whose `metadata.name` and `metadata.namespace` are required to match the associated pod,
ClusterLogs has an additional `selector` field for specifying the target pods to be simulated.
`matchNamespaces` and `matchNames` are both represented as list，which are designed to take pod collections by different levels:

1. If `matchNamespaces` is empty, ClusterLogs will be applied to all pods that are managed by `kwok` and whose names listed in `matchNames`.
2. If `matchNames` is empty, ClusterLogs will be applied to all pods managed by `kwok` and under namespaces listed in `matchNamespaces`.
3. If `matchNames` and `matchNamespaces` are both unset, ClusterLogs will be applied to all pods that `kwok` manages.

The `logs` field of ClusterLogs has the same semantic with the one in Logs.

## Examples

<img width="700px" src="/img/demo/logs.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[Logs]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Logs
[ClusterLogs]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterLogs
