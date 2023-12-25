---
title: "Stages"
---

# Stages Configuration

{{< hint "info" >}}

This document walks you through how to configure the Stages of Lifecycle.

{{< /hint >}}

## What is a Stage?

The [Stage API] is a [`kwok` Configuration][configuration] that allows users to define and simulate different stages in the lifecycle of Kubernetes resources, such as nodes and pods.
Each Stage resource specifies a resourceRef field that identifies the type of resource that the stage applies to, and a selector field that determines when the stage should be executed.

A Stage resource has the following fields:

``` yaml
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  resourceRef:
    apiGroup: <string>
    kind: <string>
  selector:
    matchLabels:
      <string>: <string>
    matchAnnotations:
      <string>: <string>
    matchExpressions:
    - key: <expressions-string>
      operator: <string>
      values:
      - <string>
  delay:
    durationMilliseconds: <int>
    durationFrom:
      expressionFrom: <expressions-string>
    jitterDurationMilliseconds: <int>
    jitterDurationFrom:
      expressionFrom: <expressions-string>
  next:
    statusTemplate: <string>
    finalizers:
      add:
      - value: <string>
      remove:
      - value: <string>
      empty: <bool>
    delete: <bool>
  immediateNextStage: <bool>
```

By setting the `selector` and `next` fields in the spec section of a Stage resource,
users can specify the conditions that need to be met for the stage to be applied,
and the changes that will be made to the resource when the stage is applied.
The `next` field allows users to define the new state of the resource using the `statusTemplate` field,
modify the `finalizers` of the resource, and even `delete` the resource.

Additionally, the `delay` field in a Stage resource allows users to specify a delay before the stage is applied,
and introduce jitter to the delay to specify the latest delay time to make the simulation more realistic.
This can be useful for simulating real-world scenarios where events do not always happen at the same time.

By configuring the `delay`, `selector`, and `next` fields in a Stage, you can control when and how the stage is applied,
providing a flexible and scalable way to simulate real-world scenarios in your Kubernetes cluster.
This allows you to create complex and realistic simulations for testing, validation, and experimentation,
and gain insights into the behavior and performance of your applications and infrastructure.

## Expressions string

The `<expressions-string>` is provided by the [Go Implementation] of [JQ Expressions]

## How `kwok` applies Stages

Stages can be generally divided into two categories based on different settings of the `next` field.
A Stage that has a non-empty `nextTemplate` is a "Change Stage". The content of `nextTemplate` will be used by `kwok` for updating resource status.
A Stage with `delete` being `true` represents a "Delete Stage", which signals `kwok` to delete the resource.

It is the [Resource Lifecycle Simulation Controller] in `kwok` that applies the Stages. The controller watches resource events from the apiserver and applies a Stage on a resource when it receives an associated event.
Let’s take a particular resource as an example. Starting from receiving an `Added` event of the resource, `kowk` checks whether the associated object matches a Stage. `kwok` then updates the resource status
if a "Change Stage" is matched. The update action itself consequently causes a new `Modified` to be generated, which will be caught by the controller later and trigger the next check and apply round.
`kwok` deletes the resource until a "Delete Stage” is matched.

However, this event-driven approach to applying Stages has a limitation: `kwok` won’t apply a Stage until a new event associated with that resource is received. To address the limitation,
users can utilize the `immediateNextStage` field to make the controller apply Stages immediately rather than waiting for an event pushed from the apiserver.

## Examples

### Node Stages

This example shows how to configure the simplest and fastest stages of Node resource, which is also the default Node stages for `kwok`.

[Default Node Stages]

### Pod Stages

This example shows how to configure the simplest and fastest stages of Pod resource, which is also the default Pod stages for `kwok`.

[Default Pod Stages]

``` goat { height=510 width=550 }
      o
      |
      | Pod scheduled to Node that managed by kwok
      v
 .---------.
| pod-ready |
 '----+----'
      |
      +
     / \
 No /   \ Yes
 .-+ Job?+-.
|   \   /   |
|    \ /    |
|     +     |
|           v
|     .------------.
|    | pod-complete |
|     '-----+------'
|           |
 '---. .---'
      |
      | .metadata.deletionTimestamp be set
      v
 .----------.
| pod-delete |
 '----+-----'
      |
      | Pod be deleted
      v
      o
```

<img width="700px" src="/img/demo/stages-pod-fast.svg">

### Pod Stages that simulate real behavior as closely as possible

[General Pod Stages]

<img width="700px" src="/img/demo/stages-pod-general.svg">

[configuration]: {{< relref "/docs/user/configuration" >}}
[Go Implementation]: https://github.com/itchyny/gojq
[JQ Expressions]: https://stedolan.github.io/jq/manual/#Basicfilters
[Default Node Stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/node/fast
[Default Pod Stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/pod/fast
[General Pod Stages]: https://github.com/kubernetes-sigs/kwok/tree/main/kustomize/stage/pod/general
[Stage API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Stage
[Resource Lifecycle Simulation Controller]: {{< relref "/docs/design/architecture" >}}
