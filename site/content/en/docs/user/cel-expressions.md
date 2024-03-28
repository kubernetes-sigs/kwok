---
title: "CEL Expressions"
---

# CEL Expressions in `kwok`

The page provides a concise note on writing CEL expressions in `kwok` CRs.  

Below is the list of all CRs in `kwok` that contains CEL based fields.
* [Metric]
* [ResourceUsage]
* [ClusterResourceUsage]


You must follow [the CEL language specification] when writing the expressions.
For predefined functions of CEL, please refer to [CEL predefined functions].

Besides the built-in functions, `kwok` also provides some customized extension functions.
An exhaustive list of all the extension functions with their usages is given below.

* `Now()`: takes no parameters and returns the current timestamp.
* `Rand()`: takes no parameters and returns a random `float64` value.
* `SinceSecond()` returns the seconds elapsed since a given resource (`pod` or `node`) was created.
  For example: `SinceSecond(pod)`, `node.SinceSecond(node)`.
* `UnixSecond()` returns the Unix time of a given time of type `time.Time`.
  For example: , `UnixSecond(Now())`, `UnixSecond(node.metadata.creationTimestamp)`.
* `Quantity()` returns a float64 value of a given Quantity value. For example: `Quantity("100m")`, `Quantity("10Mi")`.
* `Usage()` returns the current instantaneous resource usage with the simulation data in [ResourceUsage (ClusterResourceUsage)].
  For example: `Usage(pod, "memory")`, `Usage(node, "memory")`, `Usage(pod, "memory", container.name)` return the
  current working set of a resource (pod, node or container) in bytes.
* `CumulativeUsage()` returns the cumulative resource usage in seconds with the simulation data given in [ResourceUsage (ClusterResourceUsage)].
  For example: `CumulativeUsage(pod, "cpu")`, `CumulativeUsage(node, "cpu")`, `CumulativeUsage(pod, "cpu", container.name)`
  return a cumulative cpu time consumed by a resource (pod, node or container) in core-seconds.

Additionally, `kwok` provides three special CEL variables `node`, `pod`, and `container` that could be used 
in the expressions.
The three variables are set to the corresponding node, pod, container resource object respectively and users can
reference any nested fields of the resource objects simply via the CEL field selection expression (`e.f` format). 
For example, you could use expression `node.metadata.name` to obtain the node name. 

{{< hint "info" >}}

The functions with at least one parameter can be called in a receiver call-style.
That is, a function call like `f(e1, e2)` can also be called in style `e1.f(e2)`. For example, you can use `pod.Usage("memory")`
as an alternative to `Usage(pod, "memory")`.

{{< /hint >}}


It is worth noting that the use of some extension functions is restricted to specific CRs and contexts in the sense
that they are not generic but designed for special evaluating tasks.
The detailed limitations are described below.

## Functions Limitation

Function `Usage()` and `CumulativeUsage()` can only be used in the Metric resource.
For other functions listed above, users are also allowed to use them in ResourceUsage and ClusterResourceUsage
to build dynamic resource usage patterns.

The reason behind is that when `kwok` evaluates functions `Usage()` or `CumulativeUsage()`,
it actually takes the simulation data given in ResourceUsage and ClusterResourceUsage to obtain metric values.
Therefore, please ensure that the associated ResourceUsage or ClusterResourceUsage with the needed resource types
(cpu or memory) are also provided when using function `Usage()` and `CumulativeUsage()`.

## Variables Limitation

When using the three special CEL variables `node`, `pod`, and `container` in Metric resource, you should follow the below rules.
* When `dimension` is `node`: only `node` variable can be used.
* When `dimension` is `pod`: only `node`, `pod` can be used.
* When `dimension` is `container`: `node`, `pod`, `container` all can be used.


[Metric]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Metric
[ResourceUsage]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ResourceUsage
[ClusterResourceUsage]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage
[the CEL language specification]: https://github.com/google/cel-spec/blob/master/doc/langdef.md
[CEL predefined functions]: https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions
