# Create a Cluster with `kwokctl`

{{< hint "info" >}}

This document walks you through how to run `kwokctl` to manage fake clusters.

{{< /hint >}}

## Install `kwokctl`

[Install `kwokctl`]({{< relref "/docs/user/install" >}}) in your environment.

## Create a Cluster

Let's start by creating a cluster

``` console
$ kwokctl create cluster --name=kwok
kwokctl create cluster
Creating cluster "kwok-kwok"
Starting cluster "kwok-kwok"
Cluster "kwok-kwok" is ready
You can now use your cluster with:

    kubectl config use-context kwok-kwok

Thanks for using kwok!
```

And then we switch the context

``` bash
kubectl config use-context kwok-kwok
```

Subsequent usage is just like any other Kubernetes cluster

## Get Clusters

Get the clusters managed by `kwokctl`

```console
$ kwokctl get clusters
kwok
```

## Delete a Cluster

``` console
$ kwokctl delete cluster --name=kwok
Stopping cluster "kwok-kwok"
Deleting cluster "kwok-kwok"
Cluster "kwok-kwok" deleted
```

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
