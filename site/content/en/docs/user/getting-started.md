# Getting Started

{{< hint "info" >}}

This document walks you through how you can get started with KWOK easily.

{{< /hint >}}

Getting started with an open project like KWOK can be a great way to learn more
about Kubernetes. Here are some tips to help you get started.

## Installation

- [Install with Homebrew]({{< relref "/docs/user/install#homebrew" >}}) - Applicable to your local Linux/MacOS
- [Install binary releases]({{< relref "/docs/user/install#binary-releases" >}})

## Basic Usage

- [`kwok` Manages Nodes and Pods]({{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}) - Basic operations of `kwok` to manage Nodes and Pods
- `kwok` - maintain Nodes heartbeat and Pods status.
    - [`kwok` in Cluster]({{< relref "/docs/user/kwok-in-cluster" >}}) - Installing `kwok` in a cluster
    - [`kwok` out of Cluster]({{< relref "/docs/user/kwok-out-cluster" >}}) - Run `kwok` out of your cluster
- `kwokctl` - cluster creation, etcd snapshot, etc.
    - [`kwokctl` Manages Clusters]({{< relref "/docs/user/kwokctl-manage-cluster" >}}) - Create/Delete a cluster where all nodes are managed by `kwok`
    - [`kwokctl` Snapshots Cluster]({{< relref "/docs/user/kwokctl-snapshot" >}}) - Save/Restore the Etcd data of a cluster created by `kwokctl`
- [All in One Image]({{< relref "/docs/user/all-in-one-image" >}}) - Create a cluster with an all-in-one image easily

## Configuration

If any special concerns, you can configure KWOK with options and stages.

- [Options]({{< relref "/docs/user/configuration" >}})
- [Stages]({{< relref "/docs/user/stages-configuration" >}})

I hope this helps you get started with KWOK! Good luck and have fun contributing to the project!
