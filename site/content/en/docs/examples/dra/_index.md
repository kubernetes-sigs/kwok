---
title: "DRA"
---

# DRA (Dynamic Resource Allocation)

More information about DRA can be found at [here](https://kubernetes.io/docs/concepts/scheduling-eviction/dynamic-resource-allocation/).

`kwok` provides built-in, independently installable DRA driver simulations:

- [Simulate GPUs] — a fake GPU driver named `gpu.kwok.x-k8s.io`.
- [Simulate CPUs] — emulates [dra-driver-cpu]'s `dra.cpu` driver.
- [Simulate NVIDIA GPUs] — emulates [dra-driver-nvidia-gpu]'s `gpu.nvidia.com` driver.
- [Simulate Google TPUs] — emulates [dra-driver-google-tpu]'s `tpu.google.com` driver.

Each installs a DeviceClass and a Stage that publishes a ResourceSlice of fake devices
for every annotated node. Once the ResourceSlices are published, allocation and
reservation of ResourceClaims are handled natively by the kube-scheduler and
kube-controller-manager.

## Setup Cluster

DRA is enabled by default since Kubernetes 1.34, so no feature gates are needed
(the built-in stages use `resource.k8s.io/v1`, which requires Kubernetes 1.34 or later).
The simulations ship as Stage resources, so the cluster needs the Stage CRD enabled:

```bash
kwokctl create cluster --enable-crds=Stage
```

[Simulate GPUs]: {{< relref "/docs/examples/dra/gpu" >}}
[Simulate CPUs]: {{< relref "/docs/examples/dra/cpu" >}}
[Simulate NVIDIA GPUs]: {{< relref "/docs/examples/dra/nvidia-gpu" >}}
[Simulate Google TPUs]: {{< relref "/docs/examples/dra/google-tpu" >}}
[dra-driver-cpu]: https://github.com/kubernetes-sigs/dra-driver-cpu
[dra-driver-nvidia-gpu]: https://github.com/kubernetes-sigs/dra-driver-nvidia-gpu
[dra-driver-google-tpu]: https://github.com/kubernetes-sigs/dra-driver-google-tpu
