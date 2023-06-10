---
title: Exec
---

# Exec Configuration

{{< hint "info" >}}

This document walks you through how to configure the Exec feature.

{{< /hint >}}

## What is a Exec?

The [Exec API] is a [`kwok` Configuration][configuration] that allows users to define and simulate exec to Pod(s).

A Exec resource has the following fields:

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

To exec a container, you can set the `execs` field in the spec section of a Exec resource.
The `containers` field is used to match an item in the `execs` field. If the `containers` field is not set, the `execs` item will default to all containers.
The `local` field specifies the local environment to be executed.
The `workDir` field specifies the working directory of the local environment. If the `workDir` field is not set, the working directory will be the root directory.
The `envs` field specifies the environment variables of the local environment.

### ClusterExec

The [ClusterExec API] is a special Exec API which is cluster-side.

A ClusterExec resource has the following fields:

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

The `selector` field specifies the Pods to be executed.
The `matchNamespaces` field specifies the namespaces to be matched. If the `matchNamespaces` field is not set, the `ClusterExec` will match all namespaces.
The `matchNames` field specifies the names to be matched. If the `matchNames` field is not set, the `ClusterExec` will match all names.

## Examples

<img width="700px" src="/img/demo/exec.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[Exec API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Exec
[ClusterExec API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterExec
