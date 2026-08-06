---
title: "NVIDIA GPU"
---

# Simulate NVIDIA GPUs

Simulates [dra-driver-nvidia-gpu] full GPU allocation with the [NVIDIA GPU simulation stages]:
one device per GPU, published under the real driver's name `gpu.nvidia.com` with
its device attributes (`type`, `uuid`, `productName`, `brand`, `architecture`,
`cudaComputeCapability`, `driverVersion`, `cudaDriverVersion`) and `memory` capacity,
so ResourceClaims written for the real driver work unchanged.
MIG, VFIO passthrough, and ComputeDomains are not simulated.
See [Setup Cluster] first.

## Install the NVIDIA GPU Simulation Stages

```bash
kubectl apply -k "https://github.com/kubernetes-sigs/kwok/kustomize/stage/dra/nvidia-gpu"
```

## Create NVIDIA GPU Node

Annotate a fake node with `kwok.x-k8s.io/dra-nvidia-gpu` set to the number of fake GPUs.
Optionally, the `kwok.x-k8s.io/dra-nvidia-gpu-memory` annotation sets the memory of each
GPU (defaults to `80Gi`). For example, reusing the node manifest from [Create GPU Node]
with these annotations instead:

```yaml
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
    kwok.x-k8s.io/dra-nvidia-gpu: "8"
```

## Check Resource Slice

The `nvidia-gpu-resource-slice-publish` Stage publishes a ResourceSlice with one device per GPU
```bash
kubectl get resourceslice kwok-node-0-gpu.nvidia.com
NAME                         NODE          DRIVER           POOL          AGE
kwok-node-0-gpu.nvidia.com   kwok-node-0   gpu.nvidia.com   kwok-node-0   1s
```

The ResourceSlice is owned by the node, so it will be garbage collected when the node is deleted.

## Create a Pod with an NVIDIA GPU

As with the real driver, claims request devices of the `gpu.nvidia.com` DeviceClass,
and can select on the device attributes
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: single-nvidia-gpu
spec:
  devices:
    requests:
    - name: gpu
      exactly:
        deviceClassName: gpu.nvidia.com
        selectors:
        - cel:
            expression: device.attributes["gpu.nvidia.com"].architecture == "Hopper"
---
apiVersion: v1
kind: Pod
metadata:
  name: gpu-pod
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
      - name: gpu
  resourceClaims:
  - name: gpu
    resourceClaimName: single-nvidia-gpu
EOF
```

## Check Resource Claim

To check the status of the ResourceClaim
```bash
kubectl get resourceclaim single-nvidia-gpu
NAME                STATE                AGE
single-nvidia-gpu   allocated,reserved   1m
```

[NVIDIA GPU simulation stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/dra/nvidia-gpu
[dra-driver-nvidia-gpu]: https://github.com/kubernetes-sigs/dra-driver-nvidia-gpu
[Setup Cluster]: {{< relref "/docs/examples/dra#setup-cluster" >}}
[Create GPU Node]: {{< relref "/docs/examples/dra/gpu#create-gpu-node" >}}
