# Logs Configuration

{{< hint "info" >}}

This document walks you through how to configure the Logs feature.

{{< /hint >}}

## What is a Logs?

The Logs API is a [`kwok` Configuration]({{< relref "/docs/user/configuration" >}}) that allows users to define and simulate logs to Pod(s).

A Logs resource has the following fields:

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

To log a container, you can set the `logs` field in the spec section of a Logs resource.
The `containers` field is used to match an item in the `logs` field. If the `containers` field is not set, the `logs` item will default to all containers.
The `logsFile` field specifies the file path of the logs. If the `logsFile` field is not set, this item will be ignored.
The `follow` field specifies whether to follow the logs. If the `follow` field is not set, the `follow` field will default to false.

### ClusterLogs

The ClusterLogs API is a special Logs API which is cluster-side.

A ClusterLogs resource has the following fields:

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

The `selector` field specifies the Pods to be logged.
The `matchNamespaces` field specifies the namespaces to be matched. If the `matchNamespaces` field is not set, the `matchNamespaces` field will default to all namespaces.
The `matchNames` field specifies the names to be matched. If the `matchNames` field is not set, the `matchNames` field will default to all names.

## Examples

<img width="700px" src="/img/demo/logs.svg">
