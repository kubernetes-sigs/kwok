---
title: "DRA"
---

# DRA (Dynamic Resource Allocation)

More information about DRA can be found at [here](https://kubernetes.io/docs/concepts/scheduling-eviction/dynamic-resource-allocation/).

## Setup Cluster

Create a cluster with DRA feature enabled
```bash
kwokctl create cluster --kube-feature-gates="kube:DynamicResourceAllocation=true" --kube-runtime-config="resource.k8s.io/v1beta1=true"
```

## Create Device Class

Create a DeviceClass for GPU resources
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1beta1
kind: DeviceClass
metadata:
  name: kwok.x-k8s.io
spec:
  selectors:
  - cel:
      expression: device.driver == 'gpu.kwok.x-k8s'
EOF
```


## Create GPU Node

Create a fake GPU node
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
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
    kwok.x-k8s.io/gpu: 8
  capacity:
    cpu: 96
    memory: 1T
    pods: 110
    kwok.x-k8s.io/gpu: 8
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
## Create Resource Slice

Create a ResourceSlice representing a GPU device
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1beta1
kind: ResourceSlice
metadata:
  name: kwok-node-0.internal-gpu.kwok.copc4l8
spec:
  devices:
  - basic:
      attributes:
        architecture:
          string: Ada Lovelace
        brand:
          string: Kwok
        cudaComputeCapability:
          version: 8.9.0
        cudaDriverVersion:
          version: 12.9.0
        driverVersion:
          version: 575.57.8
        index:
          int: 0
        minor:
          int: 0
        productName:
          string: Kwok L4
        type:
          string: gpu
      capacity:
        memory:
          value: 23034Mi
    name: gpu-0
  driver: gpu.kwok.x-k8s
  nodeName: kwok-node-0
  pool:
    name: kwok-node-0
    resourceSliceCount: 1
EOF
```

## Create Resource Claim Template

Create a ResourceClaimTemplate for requesting a single GPU
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaimTemplate
metadata:
  name: single-gpu
spec:
  spec:
    devices:
      requests:
      - name: gpu
        deviceClassName: kwok.x-k8s.io
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