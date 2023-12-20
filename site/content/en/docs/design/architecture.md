---
title: Architecture
---

# Architecture

{{< hint "info" >}}

This document will introduce the architecture of KWOK.

{{< /hint >}}

## `kwok`

`kwok` is a resource controller, similar to `kube-controller-manager`, that is responsible for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.

It can run in any environment, use in-cluster or kubeconfig to connect to a `kube-apiserver` of cluster, and then manage the resources of the cluster.

So far, `kwok` provides two types of controllers:

- Resource Lifecycle Simulation Controller - This type of controller is used to simulate the lifecycle of Kubernetes resources. You can define and customize a resource's lifecycle through [Stages Configuration].
  It is worth noting that Resource Lifecycle Simulation Controller is just a conceptual term for document purpose and does not really exist in `kwok`. The following controllers are implemented under this type:
  * Node Controller - It is responsible for selecting the nodes to simulate, and then simulating the nodes' lifecycle, just by updating the node status field (which originally should be reported by kubelet).
  * Pod Controller - It is responsible for simulating the lifecycle of pods scheduled to those selected nodes by updating the pod status field (which originally should be reported by kubelet).
  * Stage Controller -  It is later introduced for a more general purpose, aiming to simulate the lifecycle of other Kubernetes resource types besides `v1.Node` and `v1.Pod`.
    You can think the Node Controller and the Pod Controller are the specialized implementation of the Stage Controller.
- Node Lease Controller - It is a dedicated controller in `kwok` for reporting node heartbeats by creating and renewing the node lease objects for those managed nodes. See [Node Heartbeats] and [KEP 589] for more details.

See [Stages Configuration] for more details.

## `kwokctl`

`kwokctl` is a CLI tool designed to streamline the creation and management of clusters, with nodes simulated by `kwok`.

It creates the cluster with the `kwokctl create cluster` command.

Use the runtime to start the control plane component, and then access it from `kube-apiserver` as if it was a real cluster.

``` goat { height=280 width=750 }
                                          Components

                                               |
          +--------------+---------------------------------------------------------------+
          |   etcdctl ---)--->  etcd                                       ⎫             |
          |              |       |                                         ⎪             |
          | ------------ |       |             +- kube-controller-manager  ⎪             |
          |              |       |            /                            ⎬ prometheus  |
Tools --- |   kubectl ---)-> kube-apiserver -+--- kube-scheduler           ⎪             |
          |              |                    \                            ⎪             |
          | ------------ |                     +- kwok-controller          ⎭             |
          |              +---------------------------------------------------------------+
          |   kwokctl -->|   binary   |   docker   |   podman   |   nerdctl   |   kind   |
          +--------------+---------------------------------------------------------------+
                                               |

                                           Runtimes
```

### Runtimes

We now provide some runtime to simulate the cluster, such as:

- `binary` - It will download required binaries of control plane components and start them directly.
- `docker` - It will use `docker` to start the control plane components.
- `podman` - It will use `podman` to start the control plane components.
- `nerdctl` - It will use `nerdctl` to start the control plane components.
- `kind` - It will use `kind` to start a cluster and deploy the `kwok` into it.

### Components

This is a list of control plane components that `kwokctl` will start:

- `etcd`
- `kube-apiserver`
- `kube-controller-manager`
- `kube-scheduler`
- `kwok-controller` (as `kwok`)
- `prometheus` (optional, for metrics)

### Tools

`kwokctl` provides some well-known tools as subcommands to access the cluster, such as:

- `kwokctl kubectl`
- `kwokctl etcdctl`

[Stages Configuration]: {{< relref "/docs/user/stages-configuration" >}}
[Node Heartbeats]: https://kubernetes.io/docs/reference/node/node-status/#heartbeats
[KEP 589]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/589-efficient-node-heartbeats
