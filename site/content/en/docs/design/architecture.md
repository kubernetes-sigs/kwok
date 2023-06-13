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

So far, `kwok` has implemented the following controllers:

- Node Controller - It is responsible for selecting the node to simulate, and then simulating the node's lifecycle, just by updating the heartbeat of the node.
- Pod Controller - It is responsible for pod that is on selected node, and plays the stage of pod's lifecycle.

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
