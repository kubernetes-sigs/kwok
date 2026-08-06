# DRA GPU Driver Simulation (`gpu.kwok.x-k8s.io`)

These resources simulate a DRA GPU driver named `gpu.kwok.x-k8s.io` using the Stage API,
so that DRA workflows (DeviceClass, ResourceSlice, ResourceClaim, and Pod scheduling)
can be exercised without running a real driver.

The `gpu.kwok.x-k8s.io` DeviceClass selects all devices published by the simulated driver.

The `gpu-resource-slice-publish` Stage is applied to nodes that have a `kwok.x-k8s.io/dra-gpu`
annotation whose configuration has not yet been published.
When applied, this Stage applies a ResourceSlice for the node that publishes the annotated
number of fake GPU devices (up to 128, the ResourceSlice API limit; each device optionally
reports the memory given by the `kwok.x-k8s.io/dra-gpu-memory` annotation, defaulting to
`16Gi`), and then records the published configuration in the `kwok.x-k8s.io/dra-gpu-published`
annotation on the node, so that changing any of the annotations republishes the slice.
The ResourceSlice is owned by the node, so it is garbage collected when the node is deleted.

Allocation and reservation of ResourceClaims are handled natively by the
kube-scheduler and kube-controller-manager once the ResourceSlices exist.
