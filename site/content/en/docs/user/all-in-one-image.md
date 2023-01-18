# Run cluster with all-in-one image

{{< hint "info" >}}

This document walks through the steps to create a cluster with the all-in-one image.

{{< /hint >}}

## Getting started

``` bash
docker run --rm -it -p 8080:8080 registry.k8s.io/kwok/cluster:v1.26.0
```

``` log
Creating cluster                                                                                                   cluster=kwok
Starting cluster                                                                                                   cluster=kwok
Cluster is created                                                                                      cluster=kwok elapsed=1s
You can now use your cluster with:

    kubectl config use-context kwok-kwok

Thanks for using kwok!
###############################################################################
> kubectl -s :8080 version
WARNING: This version information is deprecated and will be replaced with the output from kubectl version --short.  Use --output=yaml|json to get the full version.
Client Version: version.Info{Major:"1", Minor:"26", GitVersion:"v1.26.0", GitCommit:"b46a3f887ca979b1a5d14fd39cb1af43e7e5d12d", GitTreeState:"clean", BuildDate:"2022-12-08T19:58:30Z", GoVersion:"go1.19.4", Compiler:"gc", Platform:"linux/amd64"}
Kustomize Version: v4.5.7
Server Version: version.Info{Major:"1", Minor:"26", GitVersion:"v1.26.0", GitCommit:"b46a3f887ca979b1a5d14fd39cb1af43e7e5d12d", GitTreeState:"clean", BuildDate:"2022-12-08T19:51:45Z", GoVersion:"go1.19.4", Compiler:"gc", Platform:"linux/amd64"}
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
default           Active   0s
kube-node-lease   Active   1s
kube-public       Active   1s
kube-system       Active   1s
###############################################################################
# The above example works if your host's port is the same as the container's,
# otherwise, change it to your host's port
Starting to serve on [::]:8080
```
