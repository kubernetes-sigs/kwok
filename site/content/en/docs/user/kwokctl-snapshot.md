---
title: "Snapshot"
---

# `kwokctl` Snapshot

{{< hint "info" >}}

This document walks you through how to save and restore a cluster with `kwokctl`

{{< /hint >}}

So far, we provide two ways to save and restore clusters:

- etcd snapshot (default)
- k8s yaml

## etcd snapshot

### Save Cluster

``` bash
kwokctl snapshot save --path snapshot.db
```

### Restore Cluster

``` bash
kwokctl snapshot restore --path snapshot.db
```

## k8s yaml

We can use `--filter` to filter the resources you want to save or restore.

### Save Cluster

``` bash
kwokctl snapshot record --path cluster.yaml --snapshot
```

### Save Cluster and Record Resources Changes 

Recording continues until an interrupt signal is sent.

``` bash
kwokctl snapshot record --path cluster.yaml
```

### Restore Cluster

This way, does not delete existing resources in the cluster,
and the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl snapshot replay --path cluster.yaml --snapshot
```

### Restore Cluster and Replay Resources Changes

After the cluster snapshot is restored and the recorded resource changes are replayed.

``` bash
kwokctl snapshot replay --path cluster.yaml
```

## Export External Cluster

This like `kwokctl snapshot save --format k8s` but it will use the kubeconfig to connect to the cluster.
This is useful when you want to snapshot a cluster that is not managed by `kwokctl`.

``` bash
kwokctl snapshot export --path external-snapshot.yaml --kubeconfig /path/to/kubeconfig
```

or like that record

``` bash
kwokctl snapshot export --path external-snapshot.yaml --kubeconfig /path/to/kubeconfig --record
```

### Restore External Cluster

Let's restore the cluster we just exported.

This way, the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl create cluster
kwokctl snapshot replay --path external-snapshot.yaml --snapshot
```

or

``` bash
kwokctl create cluster
kwokctl snapshot replay --path external-snapshot.yaml
```
