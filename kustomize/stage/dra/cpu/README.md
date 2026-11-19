# DRA CPU Driver Simulation (`dra.cpu`)

These resources simulate [dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu)
using the Stage API, so that DRA workflows (DeviceClass, ResourceSlice, ResourceClaim,
and Pod scheduling) can be exercised without running the real driver.

The `dra.cpu` DeviceClass selects all devices published by the simulated driver.

The `cpu-resource-slice-publish` Stage is applied to Ready nodes that have a
`kwok.x-k8s.io/dra-cpu` annotation whose configuration has not yet been published.
Waiting for Ready mirrors a real driver, which only publishes once the node is registered,
and keeps this Stage from competing with the node lifecycle Stages. When applied, this Stage applies
a ResourceSlice for the node, and then records the published configuration in the
`kwok.x-k8s.io/dra-cpu-published` annotation on the node, so that changing any of the
annotations republishes the slice.
The ResourceSlice is owned by the node, so it is garbage collected when the node is deleted.

The CPU count must be an integer between 1 and 9999 and the NUMA count between 1 and the
CPU count, and the resulting device count must not exceed 128; nodes annotated with anything
else are ignored, leaving any previously published slice untouched. The pool holds a single
ResourceSlice that is updated in place, so `spec.pool.generation` stays constant, matching
what the upstream resourceslice controller does for single-slice pools.

Like the real driver, devices are published in one of two modes
(`kwok.x-k8s.io/dra-cpu-mode` annotation, mirroring the real driver's `cpuDeviceMode`):

- `grouped` (default): one device per NUMA node (`cpudevnumaNNN`) with
  `allowMultipleAllocations: true` and the consumable capacity `dra.cpu/cpu`,
  plus the `dra.cpu/numCPUs`, `numaNodeID`, `socketID`, `smtEnabled`, and
  cross-driver `dra.net/numaNode` attributes. Allocating from consumable capacity
  requires the `DRAConsumableCapacity` feature gate (enabled by default since
  Kubernetes 1.36).
- `individual`: one device per CPU (`cpudevNNN`) with the `dra.cpu/cpuID`, `coreID`,
  `coreType`, `cacheL3ID`, `numaNodeID`, `socketID`, `smtEnabled`, and `dra.net/numaNode`
  attributes. Limited to 128 CPUs per node by the ResourceSlice API
  (the simulation publishes a single slice per node).

The number of fake CPUs is set by the `kwok.x-k8s.io/dra-cpu` annotation, spread evenly
across the number of NUMA nodes given by the `kwok.x-k8s.io/dra-cpu-numa` annotation
(defaulting to `1`; `socketID` and `cacheL3ID` follow the NUMA node), so ResourceClaims
written for the real driver — including CEL selectors over the topology attributes and
`matchAttribute`/`distinctAttribute` constraints — work unchanged.

The published ResourceSlice carries its own desired configuration in the
`kwok.x-k8s.io/dra-cpu`, `kwok.x-k8s.io/dra-cpu-numa` and `kwok.x-k8s.io/dra-cpu-mode`
annotations, which lets the `cpu-resource-slice-repair` Stage rebuild the devices from those
annotations whenever they no longer match. This restores slices that were edited out of band,
the way a real driver reconciles its published state. A slice that is deleted outright is not
restored, since kwok Stages are not run for deletion events.

Allocation and reservation of ResourceClaims are handled natively by the
kube-scheduler and kube-controller-manager once the ResourceSlices exist.
