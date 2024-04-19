---
title: User Guide
aliases:
  - /docs/user/getting-started
---

# Getting Started

{{< hint "info" >}}

This document walks you through how you can get started with KWOK easily.

{{< /hint >}}

Getting started with an open project like KWOK can be a great way to learn more about Kubernetes.
Here are some tips to help you get started.

## Basic Usage

- [Install] - Install `kwokctl` and `kwok`
- [`kwok` Manages Nodes and Pods] - Basic operations of `kwok` to manage Nodes and Pods
- `kwok` - maintain Nodes heartbeat and Pods status.
  - [`kwok` in Cluster] - Installing `kwok` in a cluster
  - [`kwok` out of Cluster] - Run `kwok` out of your cluster
- `kwokctl` - cluster creation, etcd snapshot, etc.
  - [`kwokctl` Manages Clusters] - Create/Delete a cluster where all nodes are managed by `kwok`
  - [`kwokctl` Snapshots Cluster] - Save/Restore the Etcd data of a cluster created by `kwokctl`
- [All in One Image] - Create a cluster with an all-in-one image easily

## Configuration

If any special concerns, you can configure KWOK with the following options:

- [Configuration]
- [Stages]
- Pod Interaction
  - [PortForward]
  - [Exec]
  - [Logs]
  - [Attach]
- [Metrics]
  - [ResourceUsage]

I hope this helps you get started with KWOK! Good luck and have fun!

[Install]: {{< relref "/docs/user/installation" >}}
[`kwok` Manages Nodes and Pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[`kwok` in Cluster]: {{< relref "/docs/user/kwok-in-cluster" >}}
[`kwok` out of Cluster]: {{< relref "/docs/user/kwok-out-cluster" >}}
[`kwokctl` Manages Clusters]: {{< relref "/docs/user/kwokctl-manage-cluster" >}}
[`kwokctl` Snapshots Cluster]: {{< relref "/docs/user/kwokctl-snapshot" >}}
[All in One Image]: {{< relref "/docs/user/all-in-one-image" >}}
[Configuration]: {{< relref "/docs/user/configuration" >}}
[Stages]: {{< relref "/docs/user/stages-configuration" >}}
[PortForward]: {{< relref "/docs/user/port-forward-configuration" >}}
[Exec]: {{< relref "/docs/user/exec-configuration" >}}
[Logs]: {{< relref "/docs/user/logs-configuration" >}}
[Attach]: {{< relref "/docs/user/attach-configuration" >}}
[Metrics]: {{< relref "/docs/user/metrics-configuration" >}}
[ResourceUsage]: {{< relref "/docs/user/resource-usage-configuration" >}}
