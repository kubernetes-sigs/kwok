# DRA GPU Driver Simulation (`gpu.kwok.x-k8s.io`)

These resources simulate a DRA GPU driver named `gpu.kwok.x-k8s.io` using the Stage API,
so that DRA workflows (DeviceClass, ResourceSlice, ResourceClaim, and Pod scheduling)
can be exercised without running a real driver.

The `gpu.kwok.x-k8s.io` DeviceClass selects all devices published by the simulated driver.

The `gpu-resource-slice-publish` Stage is applied to Ready nodes that have a
`kwok.x-k8s.io/dra-gpu` annotation whose configuration has not yet been published.
Waiting for Ready mirrors a real driver, which only publishes once the node is registered,
and keeps this Stage from competing with the node lifecycle Stages.
When applied, this Stage applies a ResourceSlice for the node that publishes the annotated
number of fake GPU devices (up to 128, the ResourceSlice API limit; each device optionally
reports the memory given by the `kwok.x-k8s.io/dra-gpu-memory` annotation, defaulting to
`16Gi`), and then records the published configuration in the `kwok.x-k8s.io/dra-gpu-published`
annotation on the node, so that changing any of the annotations republishes the slice.
The ResourceSlice is owned by the node, so it is garbage collected when the node is deleted.

The device count must be an integer between 1 and 128; nodes annotated with anything else
are ignored, leaving any previously published slice untouched. The pool holds a single
ResourceSlice that is updated in place, so `spec.pool.generation` stays constant, matching
what the upstream resourceslice controller does for single-slice pools.

The published ResourceSlice carries its own desired configuration in the
`kwok.x-k8s.io/dra-gpu` and `kwok.x-k8s.io/dra-gpu-memory` annotations, which lets the
`gpu-resource-slice-repair` Stage rebuild the slice from those annotations whenever its
devices no longer match. This restores slices that were edited out of band, the way a real
driver reconciles its published state. A slice that is deleted outright is not restored,
since kwok Stages are not run for deletion events.

Allocation and reservation of ResourceClaims are handled natively by the
kube-scheduler and kube-controller-manager once the ResourceSlices exist.
