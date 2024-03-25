---
title: "Metrics"
---

# Metrics Configuration

{{< hint "info" >}}

This document walks you through how to configure the Metrics feature.

{{< /hint >}}

## What is a Metrics?

The [Metrics] is a [`kwok` Configuration][configuration] that allows users to define and simulate metrics endpoints exposed by kubelet.

The YAML below shows all the fields of a Metrics resource:

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

There are total four metric-related endpoints in kubelet: `/metrics`, `/metrics/resource`, `/metrics/probe` and `/metrics/cadvisor`,
all of which are exposed with a Prometheus style. The Metrics resource is capable of simulating endpoints with such style.

The `path` field is required and must start with `/metrics`.
To distinguish the metrics of different nodes, the path includes a variable `{nodeName}` that is replaced by the node name.

The descriptions of each sub-field are available at [Metric API][Metrics]. 
For readers' convenience, we also mirror the documents here with some additional notes.

`metrics` is a list of specific configuration items, with each corresponding to a Prometheus style metric:
* `name` defines the metric name.
* `labels` defines the metric labels, with each item corresponding to a specific metric label.
  - `name` is a const string that provides the label name.
  - `value` is represented as a [CEL expressions] that dynamically determines the label value.
    For example: you can use `node.metadata.name` to reference the node name as the label value.
* `help` defines the help string of a metric.
* `kind` defines the type of the metric: `counter`, `gauge`, or `histogram`.
* `dimension` defines where the data comes from. It could be `node`, `pod`, or `container`.
* `value` is a [CEL expressions] that defines the metric value if `kind` is `counter` or `gauge`.
* `buckets` is exclusively for customizing the data of the metric of kind `histogram`.
  - `le`, which defines the histogram bucket’s upper threshold, has the same meaning as the one of Prometheus histogram bucket.
    That is, each bucket contains values less than or equal to `le`.
  - `value` is a CEL expression that provides the value of the bucket.
  - `hidden` indicates whether to show the bucket in the metric.
    But the value of the bucket will be calculated and cumulated into the next bucket.

## Examples

Please refer to [Metrics for kubelet's `/metrics/resource` endpoint][ResourceUsage] for a detailed.

[configuration]: {{< relref "/docs/user/configuration" >}}
[Metrics]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Metrics
[CEL expressions]: {{< relref "/docs/user/cel-expressions" >}}
[ResourceUsage]: {{< relref "/docs/user/resource-usage-configuration" >}}
