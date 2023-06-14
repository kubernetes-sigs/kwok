---
title: "With Linkerd"
---

# Linkerd

More information about Linkerd can be found at [Linkerd quick start].

## Set up Cluster

``` bash
kwokctl create cluster --runtime kind
```

## Create Node

``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

## Deploy Linkerd

``` bash
linkerd check --pre
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -
```

## Migrate Controllers to Real Node

``` bash
kubectl patch deploy linkerd-destination -n linkerd --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
kubectl patch deploy linkerd-identity -n linkerd --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
kubectl patch deploy linkerd-proxy-injector -n linkerd --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Create Pod and Inject Sidecar

``` bash
kubectl apply -f https://run.linkerd.io/emojivoto.yml
kubectl get -n emojivoto deploy -o yaml | \
  linkerd inject - | \
  kubectl apply -f -
```

[Linkerd quick start]: https://linkerd.io/getting-started/
