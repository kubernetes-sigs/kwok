# Save/Restore a Cluster with `kwokctl`

{{< hint "info" >}}

This document walks you through how to save and restore a cluster with `kwokctl`

{{< /hint >}}

So far, we provide two ways to save and restore clusters:

- etcd snapshot (default)
- k8s yaml

## etcd snapshot

### Save cluster

``` bash
kwokctl snapshot save --path snapshot.db
```

### Restore cluster

``` bash
kwokctl snapshot restore --path snapshot.db
```

## k8s yaml

We can use `--filter` to filter the resources you want to save or restore.

### Save cluster

``` bash
kwokctl snapshot save --path cluster.yaml --format k8s
```

### Restore cluster

This way does not delete existing resources in the cluster,
It is a wrapper around `kubectl apply`, which will handle the ownerReference so that the resources remain relative to each other

``` bash
kwokctl snapshot restore --path cluster.yaml --format k8s
```
