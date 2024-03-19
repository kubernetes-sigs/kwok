---
title: Exec
---

# Exec Configuration

{{< hint "info" >}}

This document walks you through how to configure the Exec feature.

{{< /hint >}}

## What is an Exec?

The [Exec] is a [`kwok` Configuration][configuration] that allows users to define and simulate exec to a single pod.

The YAML below shows all the fields of an Exec resource:

``` yaml
kind: Exec
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
  namespace: <string>
spec:
  execs:
  - containers:
    - <string>
    local:
      workDir: <string>
      envs:
      - name: <string>
        value: <string>
```

To associate an Exec with a certain pod to be simulated, users must ensure `metadata.name` and `metadata.namespace` 
are inconsistent with the name and namespace of the target pod.

The exec simulation setting of a pod are specified via `execs` field.
The `execs` field is organized by groups, with each corresponding to a collection of containers that shares a same exec simulation setting.
Each group consists of a list of container names (`containers`) and the shared exec simulation setting (`local`).

{{< hint "info" >}}
If `containers` is not given in a group, the `usage` in that group will be applied to all containers of the target pod.
{{< /hint >}}

The `local` field specifies the local environment to be executed.
The `workDir` field specifies the working directory of the local environment. If not set, the working directory will be the root directory.
The `envs` field specifies the environment variables of the local environment.

### ClusterExec

In addition to simulating a single pod, users can also simulate the resource usage for multiple pods via [ClusterExec].

The YAML below shows all the fields of a ClusterExec resource:

``` yaml
kind: ClusterExec
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  selector:
    matchNamespaces:
    - <string>
    matchNames:
    - <string>
  execs:
  - containers:
    - <string>
    local:
      workDir: <string>
      envs:
      - name: <string>
        value: <string>
```

Compared to Exec, whose `metadata.name` and `metadata.namespace` are required to match the associated pod,
ClusterExec has an additional `selector` field for specifying the target pods to be simulated.
`matchNamespaces` and `matchNames` are both represented as list，which are designed to take pod collections by different levels:

1. If `matchNamespaces` is empty, ClusterExec will be applied to all pods that are managed by `kwok` and whose names listed in `matchNames`.
2. If `matchNames` is empty, ClusterExec will be applied to all pods managed by `kwok` and under namespaces listed in `matchNamespaces`.
3. If `matchNames` and `matchNamespaces` are both unset, ClusterExec will be applied to all pods that `kwok` manages.

The `execs` field of ClusterExec has the same semantic with the one in Exec.

## Examples

<img width="700px" src="/img/demo/exec.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[Exec]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Exec
[ClusterExec]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterExec
