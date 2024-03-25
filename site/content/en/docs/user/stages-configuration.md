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

`statusTemplate` and `delete` are the two fundamental fields in `next` that respectively represent the two basic phases of resource lifecycle simulation: status update and resource deletion.
`statusTemplate` provides a way to define resource status based on go template rendering. Please see
[go template in `kwok`] for more detailed instructions.
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
Please refer to [How Delay is Calculated] for more details.

By configuring the `delay`, `selector`, and `next` fields in a Stage, you can control when and how the stage is applied,
providing a flexible and scalable way to simulate real-world scenarios in your Kubernetes cluster.
This allows you to create complex and realistic simulations for testing, validation, and experimentation,
and gain insights into the behavior and performance of your applications and infrastructure.

## Expressions string

The `<expressions-string>` is provided by the [Go Implementation] of [JQ Expressions]

## How it works

Stages can be generally divided into two categories based on different settings of the `next` field.
A Stage that has a non-empty `statusTemplate` is a "Change Stage", which will be used by `kwok` for updating resource status.
A Stage with `delete` being `true` represents a "Delete Stage", which means to `kwok` to delete the resource.

It is the [Resource Lifecycle Simulation Controller] in `kwok` that applies the Stages. The controller watches resource events from the apiserver and applies a Stage on a resource when it receives an associated event.
Let’s take a particular resource as an example. Starting from receiving an `Added` event of the resource, `kwok` checks whether the associated object matches a Stage. `kwok` then updates the resource status
if a "Change Stage" is matched. The update action itself consequently causes a new `Modified` to be generated, which will be caught by the controller later and trigger the next check and apply round.
`kwok` deletes the resource until a "Delete Stage” is matched.

```goat { height=600 width=550 }
         o
         |
         |  An event comes
         +
        / \         
       /   \        
      /     \  
     /       \  Yes    .---------------.
    + "Delete"+------>+ Do some cleanup +----> o
     \ event?/         '---------------'
      \     /
       \   /
        \ / 
         +
         |
         | No
         +
        / \         
       /   \        
      /     \  
     / Match \  No
    +   any   +----> o
     \ Stage?/
      \     /
       \   /
        \ / 
         +
         |
         | Yes               
         v                    
 .---------------.         
| Modify resource |
 '-------+-------'
         |
         |  A new event will be generated
         v
         o
```

However, this event-driven approach to applying Stages has a limitation: `kwok` won’t apply a Stage until a new event associated with that resource is received. To address the limitation,
users can utilize the `immediateNextStage` field to make the controller apply Stages immediately rather than waiting for an event pushed from the apiserver.

## How Delay is Calculated

The delay time of applying a Stage is obtained by adding a constant time period and a randomized interval,
which will be denoted by *duration* and *jitter* respectively in the following.

- `durationMilliseconds`: provides *duration*.
- `jitterDurationMilliseconds`: calculates *jitter* by *random*(`jitterDurationMilliseconds`-*duration*).
  It shall be larger than `durationMilliseconds` if you want to inject jitter to the delay.
  Otherwise, it will be used directly as the delay time without any randomization.

`kwok` also provides other fields to flexibly handle more situations. They are used to extract
and parse a RFC3339 timestamp field of a resource to help determine the delay time dynamically.

- `durationFrom`: calculates *duration* by `durationFrom`-*now*
- `jitterDurationFrom`: calculates *jitter* by *random*(`jitterDurationFrom`-*now*-*duration*),
  where *now* is the timestamp when the delay starts.

{{< hint "info" >}}
`durationFrom` and `jitterDurationFrom` have a higher priority than `durationMilliseconds`
and `jitterDurationMilliseconds` if both are set.
{{< /hint >}}

Let’s explain a little bit about the motivation behind these two advanced fields.
But before that, for a better understanding, we briefly describe how kubelet "delete" a pod from a node.

Here are the steps to remove a pod in Kubernetes:

1. Execute the command `kubectl delete pod`. The apiserver receives the deletion request but does not immediately remove the corresponding pod resource from etcd.
2. The apiserver sets the `metadata.deletionTimestamp` field to the time the request was issued plus a short period, defined by `metadata.deletionGracePeriodSeconds` (default 30s).
3. The kubelet detects a non-null `metadata.deletionTimestamp` for a pod and starts to send a `TERM` signal to the main process of the container.
4. If the `metadata.deletionTimestamp` expires before the process stops by itself, the main process is then terminated using the `KILL` signal.
5. After all the containers in the pod have stopped running, the kubelet sends a force deletion request to the apiserver.
6. The apiserver removes the pod object from etcd.

To simulate this situation, you can set `jitterDurationFrom` of a "Delete Stage" (`next.delete: true`) to point to `metadata.deletionTimestamp`.
This will cause the deletion operation to occur at a random moment before `metadata.deletionTimestamp` expires.
You can also let `kwok` perform the deletion in a deterministic way by pointing `durationFrom` to `metadata.deletionTimple`,
making the deletion happen exactly at `metadata.deletionTimple`.

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
[How Delay is Calculated]: {{< relref "/docs/user/stages-configuration#how-delay-is-calculated" >}}
[go template in `kwok`]: {{< relref "/docs/user/go-template" >}}
