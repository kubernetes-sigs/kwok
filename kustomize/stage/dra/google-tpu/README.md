# DRA Google TPU Driver Simulation (`tpu.google.com`)

These resources simulate [dra-driver-google-tpu](https://github.com/kubernetes-sigs/dra-driver-google-tpu)
using the Stage API, so that DRA workflows (DeviceClass, ResourceSlice, ResourceClaim,
and Pod scheduling) can be exercised without running the real driver.

One device is published per TPU chip under the real driver's name `tpu.google.com`
with its device attributes (`index`, `tpuGen`, and `uuid` — like the real driver,
all devices of a node share one node-seeded UUID), so ResourceClaims written for
the real driver work unchanged.

The `tpu.google.com` DeviceClass matches the real driver's chart: it selects all
devices published by the driver.

The `google-tpu-resource-slice-publish` Stage is applied to nodes that have a
`kwok.x-k8s.io/dra-google-tpu` annotation whose configuration has not yet been
published. When applied, this Stage applies a ResourceSlice for the node that
publishes the annotated number of fake TPU devices (up to 128, the ResourceSlice
API limit; named `accelN`, with the TPU generation given by the
`kwok.x-k8s.io/dra-google-tpu-gen` annotation, defaulting to `v4`), and then
records the published configuration in the `kwok.x-k8s.io/dra-google-tpu-published`
annotation on the node, so that changing any of the annotations republishes the slice.
The ResourceSlice is owned by the node, so it is garbage collected when the node is deleted.

The device count must be an integer between 1 and 128; nodes annotated with anything else
are ignored, leaving any previously published slice untouched. Each republish bumps
`spec.pool.generation`, tracked by the `kwok.x-k8s.io/dra-google-tpu-generation` annotation.

Allocation and reservation of ResourceClaims are handled natively by the
kube-scheduler and kube-controller-manager once the ResourceSlices exist.
