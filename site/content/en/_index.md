---
title: Home
type: docs
---

# `KWOK` (`K`ubernetes `W`ith`O`ut `K`ubelet)

<img align="right" width="180px" src="/favicon.svg">

[KWOK] is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employs
a pretty low resource footprint that you can easily play around on your laptop.

So far we provide two tools:

- **kwok:** Core of this repo. It simulates thousands of fake Nodes.
- **kwokctl:** A CLI to facilitate creating and managing clusters simulated by Kwok.

## Getting Started

The following examples are tested to work with the latest version of Kwok/Kwokctl.

<img width="700px" src="/manage-clusters.svg">

### Basic Usage

- [Kwok Manage Nodes and Pods]({{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}) - Kwok's basic management of Node and Pod operations
- Kwok - maintain Nodes heartbeat and Pods status.
    - [Kwok in Cluster]({{< relref "/docs/user/kwok-in-cluster" >}}) - Installing Kwok in a cluster
    - [Kwok in Local]({{< relref "/docs/user/kwok-in-local" >}}) - Run Kwok in the local for a cluster
- Kwokctl - cluster creation, etcd snapshot, etc.
    - [Kwokctl Manage Clusters]({{< relref "/docs/user/kwokctl-manage-cluster" >}}) - Create/Delete a cluster in local where all nodes are managed by Kwok
    - [Kwokctl Snapshot Cluster]({{< relref "/docs/user/kwokctl-snapshot" >}}) - Save/Restore the Etcd data of a cluster created by Kwokctl

## Kwokctl Runtime and OS Support

Runtime indicates which medium kwokctl will use to start the cluster

|         | binary | docker | kind | nerdctl | podman |
| ------- | ------ | ------ | ---- | ------- | ------ |
| Linux   | 🟢      | 🟢      | 🟢    | 🟢       | ⚫      |
| Darwin  | 🟠      | 🟢      | 🟢    | 🔴       | 🔴      |
| Windows | 🟠/🔵    | 🔵      | 🔵    | 🔴       | 🔴      |

- 🟢 Supported
- 🔴 Not supported
- 🟠 Need to use your own build of the Kubernetes binary
- 🔵 Expected support but not fully tested
- ⚫ TODO

## Community

See our own [contributor guide] and the Kubernetes [community page].

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct][code of conduct].

[KWOK]: https://sigs.k8s.io/kwok
[community page]: https://kubernetes.io/community/
[contributor guide]: {{< relref "/docs/contributing/getting-started" >}}
[code of conduct]: https://github.com/kubernetes-sigs/kwok/blob/main/code-of-conduct.md
