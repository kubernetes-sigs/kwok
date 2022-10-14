# Use Kwokctl Create a Cluster

This doc walks you through how to run `kwokctl` to manage fake clusters.

## Variables preparation

``` bash
# Kwok repository to download image from
KWOK_REPO=kubernetes-sigs/kwok
# Get latest Kwok binary
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Install Kwokctl

Firstly, we download and install Kwokctl

``` bash
wget -O kwokctl -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwokctl-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwokctl
sudo mv kwokctl /usr/local/bin/kwokctl
```

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

## Delete a Cluster

``` console
$ kwokctl delete cluster --name=kwok
Stopping cluster "kwok-kwok"
Deleting cluster "kwok-kwok"
Cluster "kwok-kwok" deleted
```
