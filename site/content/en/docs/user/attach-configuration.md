# Attach Configuration

{{< hint "info" >}}

This document walks you through how to configure the Attach feature.

{{< /hint >}}

## What is a Attach?

The Attach API is a [`kwok` Configuration]({{< relref "/docs/user/configuration" >}}) that allows users to define and simulate attaching to Pod(s).

A Attach resource has the following fields:

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

To attach a container, you can set the `attaches` field in the spec section of an Attach resource.
The `containers` field is used to match an item in the `attaches` field, and the `logsFile` field specifies the file path of the logs.
Only attach to the containers specified in the `containers` field will be attached to the `logsFile`.
If the `containers` field is not set, the `attaches` item will default to all containers.
The `logsFile` field specifies the file path of the logs. If the `logsFile` field is not set, this item will be ignored.

### ClusterAttach

The ClusterAttach API is a special Attach API which is cluster-side.

A ClusterAttach resource has the following fields:

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

The `selector` field specifies the Pods to be attached.
The `matchNamespaces` field specifies the namespaces to be matched. If the `matchNamespaces` field is not set, the `matchNamespaces` field will default to all namespaces.
The `matchNames` field specifies the names to be matched. If the `matchNames` field is not set, the `matchNames` field will default to all names.

## Examples

<img width="700px" src="/img/demo/attach.svg">
