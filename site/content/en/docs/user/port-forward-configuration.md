---
title: PortForward
---

# PortForward Configuration

{{< hint "info" >}}

This document walks you through how to configure the PortForward feature.

{{< /hint >}}

## What is a PortForward?

The [PortForward] is a [`kwok` Configuration][configuration] that allows users to define and simulate port forwarding to a single pod.

The YAML below shows all the fields of a PortForward resource:

``` yaml
kind: PortForward
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
  namespace: <string>
spec:
  forwards:
  - ports:
    - <int>
    target:
      port: <int>
      address: <string>
    command:
    - <string>
    - <string>
```
To associate a PortForward with a certain pod to be simulated, users must ensure `metadata.name` and `metadata.namespace`
are inconsistent with the name and namespace of the target pod.

The attaching setting of a pod are specified via `forwards` field.
The `forwards` field is organized by groups, with each corresponding to a collection of ports that shares a same forwarding setting.
Each group consists of a list of ports numbers (`ports`) and the shared forwarding setting (`target` and `command`).

{{< hint "info" >}}
If `ports` is not given in a group, the `target` and `command` in that group will be applied to all ports of the target pod.
{{< /hint >}}

The `target` field specifies the target address to be forwarded to. If the `command` field is set, the `target` field will be ignored.
The `command` field allows users to define the command to be executed to forward the port. The `command` is executed in the container of kwok.
The `command` should be a string array, where the first element is the command and the rest are the arguments. Also, the command should be in the container’s PATH.

### ClusterPortForward

In addition to simulating a single pod, users can also simulate the port forwarding for multiple pods via [ClusterPortForward].

The YAML below shows all the fields of a ClusterPortForward resource:

``` yaml
kind: ClusterPortForward
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  selector:
    matchNamespaces:
    - <string>
    matchNames:
    - <string>
  forwards:
  - ports:
    - <int>
    target:
      port: <int>
      address: <string>
    command:
    - <string>
    - <string>
```

Compared to PortForward, whose `metadata.name` and `metadata.namespace` are required to match the associated pod,
ClusterPortForward has an additional `selector` field for specifying the target pods to be simulated.
`matchNamespaces` and `matchNames` are both represented as list，which are designed to take pod collections by different levels:

1. If `matchNamespaces` is empty, ClusterPortForward will be applied to all pods that are managed by `kwok` and whose names listed in `matchNames`.
2. If `matchNames` is empty, ClusterPortForward will be applied to all pods managed by `kwok` and under namespaces listed in `matchNamespaces`.
3. If `matchNames` and `matchNamespaces` are both unset, ClusterPortForward will be applied to all pods that `kwok` manages.

The `forwards` field of ClusterPortForward has the same semantic with the one in PortForward.

## Examples

<img width="700px" src="/img/demo/port-forward.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[PortForward]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.PortForward
[ClusterPortForward]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterPortForward
