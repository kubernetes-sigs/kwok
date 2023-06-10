---
title: All-in-one Image
---

# Run cluster with all-in-one image

{{< hint "info" >}}

This document walks through the steps to create a cluster with the all-in-one image.

{{< /hint >}}

## Getting started

``` bash
docker run --rm -it -p 8080:8080 registry.k8s.io/kwok/cluster:v0.2.0-k8s.v1.27.1
```

``` log
Cluster is creating                                                                                     cluster=kwok
Cluster is created                                                                           cluster=kwok elapsed=0s
Cluster is starting                                                                                     cluster=kwok
Cluster is started                                                                           cluster=kwok elapsed=2s
You can now use your cluster with:

	kubectl cluster-info --context kwok-kwok

Thanks for using kwok!
###############################################################################
> kubectl -s :8080 version
WARNING: This version information is deprecated and will be replaced with the output from kubectl version --short.  Use --output=yaml|json to get the full version.
Client Version: version.Info{Major:"1", Minor:"27", GitVersion:"v1.27.1", GitCommit:"4c9411232e10168d7b050c49a1b59f6df9d7ea4b", GitTreeState:"clean", BuildDate:"2023-04-14T13:21:19Z", GoVersion:"go1.20.3", Compiler:"gc", Platform:"linux/arm64"}
Kustomize Version: v5.0.1
Server Version: version.Info{Major:"1", Minor:"27", GitVersion:"v1.27.1", GitCommit:"4c9411232e10168d7b050c49a1b59f6df9d7ea4b", GitTreeState:"clean", BuildDate:"2023-04-14T13:14:42Z", GoVersion:"go1.20.3", Compiler:"gc", Platform:"linux/arm64"}
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
NAME          STATUS   AGE
default       Active   0s
kube-system   Active   1s
###############################################################################
# The above example works if your host's port is the same as the container's,
# otherwise, change it to your host's port
Starting to serve on [::]:8080
```

## Next steps

Now, you can use `kwok` to [manage nodes and pods] in the Kubernetes cluster.

[manage nodes and pods]: {{< relref "/docs/user/kwok-manage-nodes-and-pods" >}}
