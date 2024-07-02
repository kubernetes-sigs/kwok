---
title: "Snapshot"
---

# `kwokctl` Snapshot

{{< hint "info" >}}

This document walks you through how to save and restore a cluster with `kwokctl`

{{< /hint >}}

## etcd snapshot

Save and restore clusters from etcd

<img width="700px" src="./etcd-snapshot.svg">

### Save Cluster

``` bash
kwokctl snapshot save --path snapshot.db
```

### Restore Cluster

``` bash
kwokctl snapshot restore --path snapshot.db
```

## k8s yaml snapshot

Save and restore clusters from apiserver

<img width="700px" src="./yaml-snapshot.svg">

### Save Cluster

``` bash
kwokctl snapshot record --snapshot --path cluster.yaml
```

### Restore Cluster

This way, does not delete existing resources in the cluster,
and the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl snapshot replay --snapshot --path cluster.yaml
```

### Export External Cluster

It will use the kubeconfig to connect to the cluster and export resources.
This is useful when you want to snapshot a cluster that is not managed by `kwokctl`.

``` bash
kwokctl snapshot export --path external-snapshot.yaml --kubeconfig /path/to/kubeconfig
```

#### Restore External Cluster

Let's restore the cluster we just exported.

This way, the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl create cluster
kwokctl snapshot replay --path external-snapshot.yaml --snapshot
```

## k8s yaml recording

Record and replay cluster over time from apiserver

<img width="700px" src="./yaml-recording.svg">

### Record Cluster

Press Ctrl+C to stop recording resources

``` bash
kwokctl snapshot record --path recording.yaml
```

### Replay Cluster

``` bash
kwokctl snapshot replay --path recording.yaml
```

### Export External Cluster

``` bash
kwokctl snapshot export --path external-recording.yaml --record --kubeconfig /path/to/kubeconfig
```

#### Replay External Cluster

``` bash
kwokctl create cluster
kwokctl snapshot replay --path external-recording.yaml
```
