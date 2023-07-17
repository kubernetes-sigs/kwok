---
title: "With Argo"
---

# Argo

More information about Argo can be found at [Argo quick start].

## Custom Pod Behavior

The Argo Workflow is a custom resource for creating Pods, not using Job.
So we need to change the behavior of the Pod to make it work.

``` bash
wget https://raw.githubusercontent.com/kubernetes-sigs/kwok/release-0.3/stages/pod-fast.yaml
sed 's/Job/Workflow/g' pod-fast.yaml > workflow-fast.yaml
```

## Set up Cluster

``` bash
kwokctl create cluster --runtime kind -c workflow-fast.yaml
```

## Create Node

``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

## Deploy Argo

``` bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v3.4.8/install.yaml
```

## Migrate Controllers to Real Node

``` bash
kubectl patch deploy argo-server -n argo --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
kubectl patch deploy workflow-controller -n argo --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Test Workflow

``` bash
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml
```

[Argo quick start]: https://argoproj.github.io/argo-workflows/quick-start/
