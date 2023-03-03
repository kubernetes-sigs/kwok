---
title: Home
type: docs
---

# `KWOK` (`K`ubernetes `W`ith`O`ut `K`ubelet)

<img align="right" width="180px" src="/favicon.svg">

[KWOK] is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employs
a pretty low resource footprint that you can easily play around on your laptop.

## What is KWOK?

KWOK stands for Kubernetes WithOut Kubelet. So far, it provides two tools:

- `kwok` is the cornerstone of this project, responsible for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.
- `kwokctl` is a CLI tool designed to streamline the creation and management of clusters, with nodes simulated by `kwok`.

## Getting Started

The following examples are tested to work with the latest version of `kwok`/`kwokctl`.

<img width="700px" src="/manage-clusters.svg">

### Basic Usage

- [`kwok` Manages Nodes and Pods]({{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}) - Basic operations of `kwok` to manage Nodes and Pods
- `kwok` - maintain Nodes heartbeat and Pods status.
    - [`kwok` in Cluster]({{< relref "/docs/user/kwok-in-cluster" >}}) - Installing `kwok` in a cluster
    - [`kwok` in Local]({{< relref "/docs/user/kwok-in-local" >}}) - Run `kwok` in the local for a cluster
- `kwokctl` - cluster creation, etcd snapshot, etc.
    - [`kwokctl` Manages Clusters]({{< relref "/docs/user/kwokctl-manage-cluster" >}}) - Create/Delete a cluster in local where all nodes are managed by `kwok`
    - [`kwokctl` Snapshots Cluster]({{< relref "/docs/user/kwokctl-snapshot" >}}) - Save/Restore the Etcd data of a cluster created by `kwokctl`

### Contributing

- [Contributing]({{< relref "/docs/contributing/getting-started" >}}) - How to contribute to KWOK

## `kwokctl` Runtime and OS Support

Runtime indicates which medium `kwokctl` will use to start the cluster

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

### Getting Involved

If you're interested in participating in future discussions or development related to KWOK, there are several ways to get involved:

- Slack: [#kwok] for general usage discussion, [#kwok-dev] for development discussion. (visit [slack.k8s.io] for a workspace invitation)
- Open Issues/PRs/Discussions in [sigs.k8s.io/kwok]

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct][code of conduct].

[KWOK]: https://sigs.k8s.io/kwok
[community page]: https://kubernetes.io/community/
[contributor guide]: {{< relref "/docs/contributing/getting-started" >}}
[code of conduct]: https://github.com/kubernetes-sigs/kwok/blob/main/code-of-conduct.md
[sigs.k8s.io/kwok]: https://sigs.k8s.io/kwok/
[#kwok]: https://kubernetes.slack.com/messages/kwok/
[#kwok-dev]: https://kubernetes.slack.com/messages/kwok-dev/
[slack.k8s.io]: https://slack.k8s.io/
