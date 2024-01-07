---
title: "Stages"
---

# Stages Configuration

{{< hint "info" >}}

This document walks you through how to configure the Stages of Lifecycle.

{{< /hint >}}

## What is a Stage?

The [Stage API] is a [`kwok` Configuration][configuration] that allows users to define and simulate different stages in the lifecycle of Kubernetes resources, such as nodes and pods.
Each Stage resource specifies a `resourceRef` field that identifies the type of resource that the stage applies to, and a `selector` field that determines when the stage should be executed.

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
  weight: <int>
  delay:
    durationMilliseconds: <int>
    durationFrom:
      expressionFrom: <expressions-string>
    jitterDurationMilliseconds: <int>
    jitterDurationFrom:
      expressionFrom: <expressions-string>
  next:
    statusTemplate: <string>
    event:
      type: <string>
      reason: <string>
      message: <string>
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
The `next` field allows users to define the new status of the resource using the `statusTemplate` field, and even `delete` the resource.

`statusTemplate` and `delete` are the two fundamental fields in `next` that respectively represent the two basic phases
of resource lifecycle simulation: status update and resource deletion. 
`delete: true` has higher priority than a non-empty `statusTemplate`, which means `kwok` will delete the resource
rather than update its status if both are set.

Apart from the two fundamental fields, there are two other fields in `next` that allow users to do
some additional actions on a simulated resource before status update or deletion. `event` allows users to emit an associated
Kubernetes event if there is something to record. `finalizers` allows users to add or remove finalizers.
Please note that both fields can exist on their own without specifying `delete` or `statusTemplate` field.
In this case, `kwok` will only send an event or modify finalizers and will not change the status or delete a resource when applying a Stage.

It is worth noting that there is no dedicated field for arranging the execution order if multiple stages of a resource type are provided.
The execution order of stages can be controlled by utilizing `selector.matchExpressions` and `next` field together.
Specifically, users can chain the stages by ensuring that `selector.matchExpressions` of a stage match the status content specified in the `next` field of a previous stage.
Please refer to [Default Pod Stages] for a detailed example.
If multiple stages of a resource type share the same `selector` setting, `kwok` will randomly choose a stage to apply for a specific resource. 
Users can also customize the probability of a stage being selected via the `weight` field.
This is useful when you want the resources under a certain type to enter different stages according to a certain probability distribution.
Please note that `weight` only takes effect among stages with same `resourceRef` and `selector` settings.

Additionally, the `delay` field in a Stage resource allows users to specify a delay before the stage is applied,
and introduce jitter to the delay to specify the latest delay time to make the simulation more realistic.
This can be useful for simulating real-world scenarios where events do not always happen at the same time.

By configuring the `delay`, `selector`, and `next` fields in a Stage, you can control when and how the stage is applied,
providing a flexible and scalable way to simulate real-world scenarios in your Kubernetes cluster.
This allows you to create complex and realistic simulations for testing, validation, and experimentation,
and gain insights into the behavior and performance of your applications and infrastructure.

## Expressions string

The `<expressions-string>` is provided by the [Go Implementation] of [JQ Expressions]

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
