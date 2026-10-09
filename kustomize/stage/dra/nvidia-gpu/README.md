# DRA NVIDIA GPU Driver Simulation (`gpu.nvidia.com`)

These resources simulate [dra-driver-nvidia-gpu](https://github.com/kubernetes-sigs/dra-driver-nvidia-gpu)
using the Stage API, so that DRA workflows (DeviceClass, ResourceSlice, ResourceClaim,
and Pod scheduling) can be exercised without running the real driver.

The simulation covers full GPU allocation: one device per GPU, published under the
real driver's name `gpu.nvidia.com` with its device attributes (`type`, `uuid`,
`productName`, `brand`, `architecture`, `cudaComputeCapability`, `driverVersion`,
`cudaDriverVersion`, and the standard `resource.kubernetes.io/numaNode`) and the
`memory` capacity, so ResourceClaims written for the real driver work unchanged.
MIG, VFIO passthrough, and ComputeDomains are not simulated.

The `gpu.nvidia.com` DeviceClass matches the real driver's chart: it selects devices
of the driver with `type == 'gpu'`, and maps the `nvidia.com/gpu` extended resource
name (used when the `DRAExtendedResource` feature gate is enabled).

The `nvidia-gpu-resource-slice-publish` Stage is applied to Ready nodes that have a
`kwok.x-k8s.io/dra-nvidia-gpu` annotation whose configuration has not yet been
published. Waiting for Ready mirrors a real driver, which only publishes once the node is
registered, and keeps this Stage from competing with the node lifecycle Stages. When applied, this Stage applies a ResourceSlice for the node that
publishes the annotated number of fake GPU devices (up to 128, the ResourceSlice
API limit; each device reports the memory given by the
`kwok.x-k8s.io/dra-nvidia-gpu-memory` annotation, defaulting to `80Gi`), and then
records the published configuration in the `kwok.x-k8s.io/dra-nvidia-gpu-published`
annotation on the node, so that changing any of the annotations republishes the slice.
The ResourceSlice is owned by the node, so it is garbage collected when the node is deleted.

The device count must be an integer between 1 and 128; nodes annotated with anything else
are ignored, leaving any previously published slice untouched. The pool holds a single
ResourceSlice that is updated in place, so `spec.pool.generation` stays constant, matching
what the upstream resourceslice controller does for single-slice pools.

The published ResourceSlice carries its own desired configuration in the
`kwok.x-k8s.io/dra-nvidia-gpu` and `kwok.x-k8s.io/dra-nvidia-gpu-memory` annotations, which
lets the `nvidia-gpu-resource-slice-repair` Stage rebuild the devices from those annotations
whenever they no longer match. This restores slices that were edited out of band, the way a
real driver reconciles its published state. A slice that is deleted outright is not restored,
since kwok Stages are not run for deletion events.

Allocation and reservation of ResourceClaims are handled natively by the
kube-scheduler and kube-controller-manager once the ResourceSlices exist.
