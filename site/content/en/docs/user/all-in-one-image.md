---
title: All-in-one Image
---

# Run cluster with all-in-one image

{{< hint "info" >}}

This document walks through the steps to create a cluster with the all-in-one image.

{{< /hint >}}

## Getting started

``` bash
docker run --rm -it -p 8080:8080 registry.k8s.io/kwok/cluster:v0.4.0-k8s.v1.28.0
```

- [ghcr.io/kwok-ci/cluster]: Daily build with the latest kwok and the latest k8s.
- [registry.k8s.io/kwok/cluster]: Build only the last 6 k8s releases in each kwok release.

### Quick Verification

You can use the `kubectl` with the `-s` option to connect to the cluster.

``` bash
kubectl -s :8080 get ns
```

``` log
NAME              STATUS   AGE
default           Active   1s
kube-node-lease   Active   1s
kube-public       Active   1s
kube-system       Active   1s
```

### Setting up kubeconfig

You can set up the `kubeconfig` file to connect to the cluster.

``` bash
apiVersion: v1
clusters:
- cluster:
    server: http://127.0.0.1:8080
  name: kwok
contexts:
- context:
    cluster: kwok
  name: kwok
current-context: kwok
kind: Config
preferences: {}
users: null
```

## Use in a pod

If you are using the all-in-one image in a pod,
you need disable the service account token or the cluster might not work properly.

``` yaml
...
spec:
  automountServiceAccountToken: false
...
```

or

remove the service account token file `/var/run/secrets/kubernetes.io/serviceaccount/token`.

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
[ghcr.io/kwok-ci/cluster]: https://github.com/kwok-ci/cluster/pkgs/container/cluster
[registry.k8s.io/kwok/cluster]: https://github.com/kubernetes/k8s.io/blob/main/registry.k8s.io/images/k8s-staging-kwok/images.yaml
