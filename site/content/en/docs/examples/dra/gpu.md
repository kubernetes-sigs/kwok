---
title: "GPU"
---

# Simulate GPUs

Simulates a fake GPU driver named `gpu.kwok.x-k8s.io` with the [GPU simulation stages].
See [Setup Cluster] first.

## Install the GPU Simulation Stages

```bash
kubectl apply -k "https://github.com/kubernetes-sigs/kwok/kustomize/stage/dra/gpu"
```

## Create GPU Node

Create a fake GPU node with the `kwok.x-k8s.io/dra-gpu` annotation set to the number of fake GPUs.
Optionally, the `kwok.x-k8s.io/dra-gpu-memory` annotation sets the memory capacity of each device (defaults to `16Gi`).

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
    kwok.x-k8s.io/dra-gpu: "8"
    kwok.x-k8s.io/dra-gpu-memory: "24Gi"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kwok-node-0
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: kwok-node-0
spec:
  taints: # Avoid scheduling actual running pods to fake Node
  - effect: NoSchedule
    key: kwok.x-k8s.io/node
    value: fake
status:
  allocatable:
    cpu: 96
    memory: 1T
    pods: 110
  capacity:
    cpu: 96
    memory: 1T
    pods: 110
  nodeInfo:
    architecture: amd64
    bootID: ""
    containerRuntimeVersion: ""
    kernelVersion: ""
    kubeProxyVersion: fake
    kubeletVersion: fake
    machineID: ""
    operatingSystem: linux
    osImage: ""
    systemUUID: ""
  phase: Running
EOF
```

## Check Resource Slice

The `gpu-resource-slice-publish` Stage publishes a ResourceSlice for the node
```bash
kubectl get resourceslice
NAME                            NODE          DRIVER              POOL          AGE
kwok-node-0-gpu.kwok.x-k8s.io   kwok-node-0   gpu.kwok.x-k8s.io   kwok-node-0   1s
```

The ResourceSlice is owned by the node, so it will be garbage collected when the node is deleted.

## Create Resource Claim Template

Create a ResourceClaimTemplate for requesting a single GPU
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: single-gpu
spec:
  spec:
    devices:
      requests:
      - name: gpu
        exactly:
          deviceClassName: gpu.kwok.x-k8s.io
EOF
```

## Create a Pod

Create a Deployment with single replica reference Resource Claim Template
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fake-pod
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fake-pod
  template:
    metadata:
      labels:
        app: fake-pod
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: In
                values:
                - kwok
      tolerations:
      - key: "kwok.x-k8s.io/node"
        operator: "Exists"
        effect: "NoSchedule"
      containers:
      - name: fake-container
        image: fake-image
        resources:
          claims:
          - name: gpu0
      resourceClaims:
      - name: gpu0
        resourceClaimTemplateName: single-gpu
EOF
```

## Check Resource Claim

To check the status of the ResourceClaim
```bash
kubectl get resourceclaim
NAME                                   STATE                AGE
fake-pod-7589f9b49f-pcjtg-gpu0-qjzpj   allocated,reserved   61m
```

## Check Pod

To check the status of the Pod
```bash
kubectl get pod
NAME                        READY   STATUS    RESTARTS   AGE
fake-pod-7589f9b49f-pcjtg   1/1     Running   0          61m
```

[GPU simulation stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/dra/gpu
[Setup Cluster]: {{< relref "/docs/examples/dra#setup-cluster" >}}
