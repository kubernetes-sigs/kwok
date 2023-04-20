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
kwokctl snapshot save --path cluster.yaml --format k8s
```

### Restore Cluster

This way, does not delete existing resources in the cluster,
and the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl snapshot restore --path cluster.yaml --format k8s
```

## Export External Cluster

This like `kwokctl snapshot save --format k8s` but it will use the kubeconfig to connect to the cluster.
This is useful when you want to snapshot a cluster that is not managed by `kwokctl`.

``` bash
kwokctl snapshot export --path external-snapshot.yaml --kubeconfig /path/to/kubeconfig
```

### Restore External Cluster

Let's restore the cluster we just exported.

This way, the `ownerReference` field of the resources is updated to re-link them with their parent resources,
so we can preserve the hierarchy and dependencies of the resources in restore.

``` bash
kwokctl create cluster
kwokctl snapshot restore --path external-snapshot.yaml --format k8s
```
