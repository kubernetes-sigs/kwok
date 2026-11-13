---
title: "CPU"
---

# Simulate CPUs

Simulates [dra-driver-cpu]'s `dra.cpu` driver with the [CPU simulation stages],
publishing its topology attributes so ResourceClaims written for the real driver
work unchanged. See [Setup Cluster] first.

Like the real driver, devices are published in one of two modes (set by the
`kwok.x-k8s.io/dra-cpu-mode` annotation):

- `grouped` (default): one device per NUMA node with the consumable capacity `dra.cpu/cpu`.
  Allocating from consumable capacity requires the `DRAConsumableCapacity` feature gate
  (enabled by default since Kubernetes 1.36).
- `individual`: one device per CPU with the `dra.cpu/cpuID`, `coreID`, `coreType`,
  `cacheL3ID`, `numaNodeID`, `socketID`, and `smtEnabled` attributes.

## Install the CPU Simulation Stages

```bash
kubectl apply -k "https://github.com/kubernetes-sigs/kwok/kustomize/stage/dra/cpu"
```

## Create CPU Node

Create a fake node with the `kwok.x-k8s.io/dra-cpu` annotation set to the number of fake CPUs.
Optionally, the `kwok.x-k8s.io/dra-cpu-numa` annotation spreads the CPUs evenly across that
number of NUMA nodes (defaults to `1`; `socketID` and `cacheL3ID` follow the NUMA node).

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
    kwok.x-k8s.io/dra-cpu: "8"
    kwok.x-k8s.io/dra-cpu-numa: "2"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kwok-node-1
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: kwok-node-1
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

The `cpu-resource-slice-publish` Stage publishes a ResourceSlice for the node
(in the default `grouped` mode, one device per NUMA node)
```bash
kubectl get resourceslice kwok-node-1-dra.cpu
NAME                  NODE          DRIVER    POOL          AGE
kwok-node-1-dra.cpu   kwok-node-1   dra.cpu   kwok-node-1   1s
```

The ResourceSlice is owned by the node, so it will be garbage collected when the node is deleted.

## Create a Pod with CPUs on a specific NUMA node

As with the real driver, CPUs are requested as `dra.cpu/cpu` capacity and a CEL
selector over the topology attributes constrains where the CPUs come from
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: cpus-on-numa0
spec:
  devices:
    requests:
    - name: cpus
      exactly:
        deviceClassName: dra.cpu
        capacity:
          requests:
            dra.cpu/cpu: "2"
        selectors:
        - cel:
            expression: device.attributes["dra.cpu"].numaNodeID == 0
---
apiVersion: v1
kind: Pod
metadata:
  name: pinned-pod
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
  - name: app
    image: fake-image
    resources:
      claims:
      - name: cpus
  resourceClaims:
  - name: cpus
    resourceClaimName: cpus-on-numa0
EOF
```

## Check Resource Claim

To check the status of the ResourceClaim
```bash
kubectl get resourceclaim cpus-on-numa0
NAME            STATE                AGE
cpus-on-numa0   allocated,reserved   1m
```

[CPU simulation stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/dra/cpu
[dra-driver-cpu]: https://github.com/kubernetes-sigs/dra-driver-cpu
[Setup Cluster]: {{< relref "/docs/examples/dra#setup-cluster" >}}
