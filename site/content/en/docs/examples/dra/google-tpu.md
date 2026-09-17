---
title: "Google TPU"
---

# Simulate Google TPUs

Simulates [dra-driver-google-tpu] with the [Google TPU simulation stages]: one device
per TPU chip (named `accelN`), published under the real driver's name `tpu.google.com`
with its device attributes (`index`, `tpuGen`, `uuid`), so ResourceClaims written for
the real driver work unchanged. See [Setup Cluster] first.

## Install the Google TPU Simulation Stages

```bash
kubectl apply -k "https://github.com/kubernetes-sigs/kwok/kustomize/stage/dra/google-tpu"
```

## Create TPU Node

Annotate a fake node with `kwok.x-k8s.io/dra-google-tpu` set to the number of fake
TPU chips. Optionally, the `kwok.x-k8s.io/dra-google-tpu-gen` annotation sets the
TPU generation (defaults to `v4`). For example, reusing the node manifest from
[Create GPU Node] with these annotations instead:

```yaml
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
    kwok.x-k8s.io/dra-google-tpu: "4"
    kwok.x-k8s.io/dra-google-tpu-gen: "v6e"
```

## Check Resource Slice

The `google-tpu-resource-slice-publish` Stage publishes a ResourceSlice with one device per TPU chip
```bash
kubectl get resourceslice kwok-node-0-tpu.google.com
NAME                         NODE          DRIVER           POOL          AGE
kwok-node-0-tpu.google.com   kwok-node-0   tpu.google.com   kwok-node-0   1s
```

The ResourceSlice is owned by the node, so it will be garbage collected when the node is deleted.

## Create a Pod with TPUs

As with the real driver, claims request devices of the `tpu.google.com` DeviceClass,
and can select on the device attributes
```bash
kubectl apply -f - <<EOF
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: tpus
spec:
  devices:
    requests:
    - name: tpus
      exactly:
        deviceClassName: tpu.google.com
        count: 4
        selectors:
        - cel:
            expression: device.attributes["tpu.google.com"].tpuGen == "v6e"
---
apiVersion: v1
kind: Pod
metadata:
  name: tpu-pod
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
      - name: tpus
  resourceClaims:
  - name: tpus
    resourceClaimName: tpus
EOF
```

## Check Resource Claim

To check the status of the ResourceClaim
```bash
kubectl get resourceclaim tpus
NAME   STATE                AGE
tpus   allocated,reserved   1m
```

[Google TPU simulation stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/dra/google-tpu
[dra-driver-google-tpu]: https://github.com/kubernetes-sigs/dra-driver-google-tpu
[Setup Cluster]: {{< relref "/docs/examples/dra#setup-cluster" >}}
[Create GPU Node]: {{< relref "/docs/examples/dra/gpu#create-gpu-node" >}}
