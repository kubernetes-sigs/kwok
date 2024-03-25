---
title: "Istio"
---

# Istio

More information about Istio can be found at [Istio quick start].

## Set up Cluster

``` bash
kwokctl create cluster --runtime kind
```

## Create Node

``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

## Deploy Istio

``` bash
istioctl install -y
```

## Migrate Controllers to Real Node

``` bash
kubectl patch deploy istiod -n istio-system --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
kubectl patch deploy istio-ingressgateway -n istio-system --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Create Pod and Inject Sidecar

``` bash
kubectl label namespace default istio-injection=enabled
kubectl apply -f https://raw.githubusercontent.com/istio/istio/master/samples/bookinfo/platform/kube/bookinfo.yaml
```

[Istio quick start]: https://istio.io/latest/docs/setup/getting-started/
