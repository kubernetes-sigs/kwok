---
title: "Metrics"
---

# Metrics Configuration

{{< hint "info" >}}

This document walks you through how to configure the Metrics feature.

{{< /hint >}}

## What is a Metrics?

The [Metrics] is a [`kwok` Configuration][configuration]  that allows users to define and simulate metrics endpoints exposed by kubelet.

A Metrics resource has the following fields:

``` yaml
kind: Metrics
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: <string>
spec:
  path: <string>
  metrics:
  - name: <string>
    help: <string>
    kind: <string>
    dimension: <string>
    labels:
    - name: <string>
      value: <string>
    value: <string>   # for counter and gauge
    buckets:          # for histogram
    - le: <float64>
      value: <string>
      hidden: <bool>
```

There are total four metric-related endpoints in kubelet: `/metrics`, `/metrics/resource`, `metrics/probe`, `metrics/cadvisor`,
all of which are exposed with a Prometheus style. The Metrics resource is capable of simulating endpoints with such style.

To simulate a metric endpoint, first, you need to specify the RESTful `path` of the endpoint,
which will be installed and exposed by the metric service of `kwok` at port `10247` after applied.
The `path` must start with `/metrics`, otherwise, `kwok` will not install it.


{{< hint "info" >}}
Starting from metrics-server 0.7.0, it is allowed to specify the path to scrape metrics for a node.
Specifically, metric-server will check if a node has annotation `metrics.k8s.io/resource-metrics-path` 
and use it as the target metric scrape path. Combing with the Metric CR, the feature makes it possible to integrate
`kwok` and metrics-server easily. For a fake node, by adding that annotation and setting its value to the `path`
specified in a Metric resource, metrics-server will collect data from the endpoints exposed by `kwok` instead of
scrapping from kubelet.
{{< /hint >}}

Besides, compared to kubelet, which only exposes the metric of the node it is located on, `kwok` needs to expose the
metrics of all the fake nodes it manages. Instead of creating a separate Metric CR for each fake node, it is possible
to bind all the metrics endpoints from different nodes into a single `path`. Metric CR allows for a built-in
`{nodeName}` path parameter to be included in the `path` field. For example: `/metrics/nodes/{nodeName}/metrics/resource`.
With `{nodeName}`, a single `path` is able to differentiate the metric data from different nodes.


The `metrics` field are used to customize the return body of the installed metrics endpoint.
`metrics` is a list of specific configuration items, with each corresponding to a Prometheus style metric:
* `name` defines the metric name.
* `labels` defines the metric labels, with each item corresponding to a specific metric label.
  - `name` is a const string that provides the label name.
  - `value` is represented as a CEL expression that dynamically determines the label value.
    For example: you can use `pod.metadata.name` to reference the pod name as the label value.
* `help` defines the help string of a metric.
* `kind` defines the type of the metric: `counter`, `guage` or `histogram`.
* `dimension` defines where the data comes from. It could be `node`, `pod`, or `container`.
* `value` is a CEL expression that defines the metric value if `kind` is `counter` or `guage`.
  Please refer to [built-in CEL extension functions] for an exhausted list that you can use to simulate the metric value.
* `buckets` is exclusively for customizing the data of the metric of kind `histogram`.
  - `le` has the same meaning as the one of Prometheus histogram bucket.
  - `value` is a CEL expression that provides the value of the bucket.
  - `hidden` indicates whether to hide the bucket. The bucket would be ignored if set to `true`.

Please refer to [Metrics for kubelet's "metrics/resource" endpoint] for a detailed example.


## built-in CEL extension functions

As `kwok` doesn't actually run containers, to simulate the resource usage data, `kwok` provides the following CEL
extension functions you can use in a CEL expression to help calculate the simulation data flexibly and dynamically:
* `Now()`: yields the current timestamp.
* `Rand()`: yields a random `float64` value.
* `SinceSecond()` yields the seconds elapsed since a resource (pod or node) was created.
  For example: `pod.SinceSecond()`, `node.SinceSecond()`.
* `UnixSecond()` yields the Unix time of a given time of type `time.Time`.
  For example: , `UnixSecond(Now())`, `UnixSecond(node.metadata.creationTimestamp)`.
* `Quantity()` yields a float64 value of a given Quantity value. For example: `Quantity("100m")`, `Quantity("10Mi")`.
* `Usage()` yields the current instantaneous resource usage with the simulation data in [ResourceUsage (ClusterResourceUsage)].
  For example: `Usage(pod, "memory")`, `Usage(node, "memory")`, `Usage(pod, "memory", container.name)` yield the
  current working set of a resource (pod, node or container) in bytes.
* `CumulativeUsage()` yield the cumulative resource usage in seconds with the simulation data given in [ResourceUsage (ClusterResourceUsage)].
  For example: `CumulativeUsage(pod, "cpu")`, `CumulativeUsage(node, "cpu")`, `CumulativeUsage(pod, "cpu", container.name)`
  yield a cumulative cpu time consumed by a resource (pod, node or container) in core-seconds.


`node`, `pod`, `container` are three special parameters that can be used only when `dimension` is set to the corresponding value.
You should follow the below rules to use the three parameters:
* When dimension is `node`: only `node` parameter can be used.
* When dimension is `pod`: only `node`, `pod` can be used.
* When dimension is `container`: `node`, `pod`, `container` all can be used.


Except `Now()`, `Rand()` and `Quantity()`, all other functions can be invoked in a receiver call-style.
That is, a function call like `f(e1, e2)` can also be invoked in style `e1.f(e2)`.  For instance, you can use `pod.Usage("memory")`
as an alternative of `Usage(pod, "memory")`.


{{< hint "warning" >}}

Function `Usage()` and `CumulativeUsage()` can only be used in the Metric resource.
For other functions listed above, users are also allowed to use them in ResourceUsage and ClusterResourceUsage
to build dynamic resource usage patterns.

The reason behind is that when `kwok` evaluates functions `Usage()` or `CumulativeUsage()`,
it actually takes the simulation data given in ResourceUsage and ClusterResourceUsage to obtain metric values.
Therefore, please ensure that the associated ResourceUsage or ClusterResourceUsage with the needed resource types
(cpu or memory) are also provided when using function `Usage()` and `CumulativeUsage()`.

{{< /hint >}}


## Out-of-box Metric Config

`kwok` currently provides the [Metrics config][Metrics for kubelet's "metrics/resource" endpoint] that is capable of
simulating kubelet's `"metrics/resource"` endpoint.

To integrate the simulated endpoint with metrics-server (version is required >= 0.7.0), please add the 
`"metrics.k8s.io/resource-metrics-path": "/metrics/nodes/<nodeName>/metrics/resource"` annotation to the fake
nodes managed by `kwok`.

[configuration]: {{< relref "/docs/user/configuration" >}}
[Metrics]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Metrics
[built-in CEL extension functions]: {{< relref "/docs/user/metrics-configuration" >}}#built-in-cel-extension-functions
[Metrics for kubelet's "metrics/resource" endpoint]: https://github.com/kubernetes-sigs/kwok/blob/main/kustomize/metrics/resource/metrics-resource.yaml
[ResourceUsage (ClusterResourceUsage)]: {{< relref "/docs/user/resource-usage-configuration" >}}
