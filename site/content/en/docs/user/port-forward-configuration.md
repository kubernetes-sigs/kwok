# PortForward Configuration

{{< hint "info" >}}

This document walks you through how to configure the PortForward feature.

{{< /hint >}}

## What is a PortForward?

The PortForward API is a [`kwok` Configuration]({{< relref "/docs/user/configuration" >}}) that allows users to define and simulate port forwarding to Pod(s).

A PortForward resource has the following fields:

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

    - ports:
        - <int>
      command:
        - <string>
        - <string>
```

To forward a port, you can set the `forwards` field in the spec section of a PortForward resource.
The `ports` field is used to match an item in the `forwards` field. If the `ports` field is not set, the `forwards` item will default to all ports.
The `target` field specifies the target address to be forwarded to. If the `command` field is set, the `target` field will be ignored.
The `command` field allows users to define the command to be executed to forward the port. The `command` is executed in the container of kwok.
The `command` should be a string array, where the first element is the command and the rest are the arguments. Also, the command should be in the container’s PATH.

### ClusterPortForward

The ClusterPortForward API is a special PortForward API which is cluster-side.

A ClusterPortForward resource has the following fields:

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

    - ports:
        - <int>
      command:
        - <string>
        - <string>
```

The `selector` field is used to select the Pods to be port forwarded.
The `matchNamespaces` field is used to match the namespace of the Pods. If the `matchNamespaces` field is not set, the ClusterPortForward will match all namespaces.
The `matchNames` field is used to match the name of the Pods. If the `matchNames` field is not set, the ClusterPortForward will match all Pods.

## Examples

<img width="700px" src="/img/demo/port-forward.svg">
