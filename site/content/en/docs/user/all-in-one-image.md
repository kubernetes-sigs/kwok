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

``` log
Cluster is creating                                                       cluster=kwok
Cluster is created                                           elapsed=0.3s cluster=kwok
Cluster is starting                                                       cluster=kwok
Cluster is started                                           elapsed=0.7s cluster=kwok
You can now use your cluster with:

	kubectl cluster-info --context kwok-kwok

Thanks for using kwok!
Starting to serve on [::]:8080
###############################################################################
> kubectl -s :8080 version
Client Version: v1.28.0
Kustomize Version: v5.0.4-0.20230601165947-6ce0bf390ce3
Server Version: v1.28.0
###############################################################################
# The following kubeconfig can be used to connect to the Kubernetes API server
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
###############################################################################
> kubectl -s :8080 get ns
NAME              STATUS   AGE
default           Active   1s
kube-node-lease   Active   1s
kube-public       Active   1s
kube-system       Active   1s
###############################################################################
# The above example works if your host's port is the same as the container's,
# otherwise, change it to your host's port
```

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
