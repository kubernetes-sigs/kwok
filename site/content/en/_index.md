---
title: Home
type: docs
---

# `KWOK` (`K`ubernetes `W`ith`O`ut `K`ubelet)

<img align="right" width="180px" src="/favicon.svg">

[kwok](https://sigs.k8s.io/kwok) is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employes
a pretty low resource footprint that you can easily play around on your laptop.

So far we provide two tools:

- **Kwok:** Core of this repo. It simulates thousands of fake Nodes.
- **Kwokctl:** A CLI to facilitate creating and managing clusters simulated by Kwok.

## Getting Started

The following examples are tested to work with the latest version of Kwok/Kwokctl.

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
| Linux   | 🟢      | 🟢      | 🟢    | 🔵       | ⚫      |
| Darwin  | 🟠      | 🟢      | 🟢    | 🔴       | 🔴      |
| Windows | 🟠/🔵    | 🔵      | 🔵    | 🔴       | 🔴      |

- 🟢 Supported
- 🔴 Not supported
- 🟠 Need to use your own build of the Kubernetes binary
- 🔵 Expected support but not fully tested
- ⚫ TODO

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](https://github.com/kubernetes-sigs/kwok/blob/main/code-of-conduct.md).
