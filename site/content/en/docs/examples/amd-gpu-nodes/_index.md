---
title: "AMD GPU Nodes"
---

# AMD GPU Nodes

{{< hint "info" >}}

This example creates fake nodes that look like nodes running the [AMD GPU Operator](https://github.com/ROCm/gpu-operator):
the labels its node labeller writes and the extended resources its device plugin advertises.
Schedulers place pods on them as on real AMD Instinct GPU nodes, so multi-GPU placement, GPU partitioning and quota logic
can be tested without the hardware.

{{< /hint >}}

## The nodes

| Node | Platform | `amd.com/gpu` | Per device |
| --- | --- | --- | --- |
| `amd-mi300x-node` | MI300X server: 8 GPUs on a universal baseboard, linked over xGMI | 8 | 192G VRAM, 304 CUs |
| `amd-mi355x-cpx-node` | MI355X server with each GPU in CPX compute and NPS2 memory partition mode | 64 | 36G VRAM, 32 CUs |
| `amd-helios-tray-node` | One compute tray of an AMD Helios rack (MI455X) | 4 | 432G VRAM |

The labels follow the AMD node labeller (`amd.com/gpu.*` and `beta.amd.com/gpu.*`, with values from `amdgpu.ids`),
and the resource is named as the AMD device plugin does with its default `resource_naming_strategy` (`single`).
A partitioned GPU shows up as one device per partition, each reporting its share of the GPU's VRAM and CUs.

The MI455X is not in `amdgpu.ids` yet, and AMD has not published what the labeller will report for its product name,
family or CU count, so the Helios tray node carries only the VRAM label.

Fake nodes have no GPUs, device plugin or partition manager: they test placement, not device injection or performance.

## Prerequisites

- KWOK must be installed on the machine. See [installation](https://kwok.sigs.k8s.io/docs/user/installation/).
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)

## Create the nodes

```bash
kwokctl create cluster
kubectl apply -f mi300x-node.yaml -f mi355x-cpx-node.yaml -f helios-tray-node.yaml
kubectl get nodes -L amd.com/gpu.product-name -L amd.com/gpu.compute-memory-partition
```

```console
NAME                   STATUS   ROLES    AGE   VERSION       GPU.PRODUCT-NAME      GPU.COMPUTE-MEMORY-PARTITION
amd-helios-tray-node   Ready    <none>   5s    kwok-v0.8.0
amd-mi300x-node        Ready    <none>   5s    kwok-v0.8.0   AMD_Instinct_MI300X   spx_nps1
amd-mi355x-cpx-node    Ready    <none>   5s    kwok-v0.8.0   AMD_Instinct_MI355X   cpx_nps2
```

## Schedule pods

A pod that asks for all eight GPUs of an MI300X server, and one that asks for a single CPX partition:

```bash
kubectl apply -f whole-gpus-pod.yaml -f partition-pod.yaml
kubectl get pods -o wide
```

```console
NAME             READY   STATUS    RESTARTS   AGE   IP         NODE                  NOMINATED NODE   READINESS GATES
partition-pod    1/1     Running   0          1s    10.0.0.2   amd-mi355x-cpx-node   <none>           <none>
whole-gpus-pod   1/1     Running   0          1s    10.0.0.1   amd-mi300x-node       <none>           <none>
```

## More nodes

The manifests were generated with [fake_gpu_nodes.py](https://github.com/openinferencefabric/oif/tree/24d3d6e/tools/fake-gpu),
a standard-library Python script that writes any number of these nodes: MI300X, MI325X, MI350X, MI355X and MI350P servers,
Helios racks of 18 trays, the SPX, DPX, QPX and CPX compute modes with the NPS memory modes each GPU supports,
and the device plugin's `mixed` and `product` resource naming (for example `amd.com/cpx_nps2`).

{{< details "mi300x-node.yaml" >}}

{{< code-sample file="mi300x-node.yaml" >}}

{{< /details >}}

{{< details "mi355x-cpx-node.yaml" >}}

{{< code-sample file="mi355x-cpx-node.yaml" >}}

{{< /details >}}

{{< details "helios-tray-node.yaml" >}}

{{< code-sample file="helios-tray-node.yaml" >}}

{{< /details >}}

{{< details "whole-gpus-pod.yaml" >}}

{{< code-sample file="whole-gpus-pod.yaml" >}}

{{< /details >}}

{{< details "partition-pod.yaml" >}}

{{< code-sample file="partition-pod.yaml" >}}

{{< /details >}}
