---
title: "Metrics"
---

# Metrics Configuration

{{< hint "info" >}}

This document walks you through how to configure the Metrics feature.

{{< /hint >}}

## What is a Metrics?

The [Metrics API] is a [`kwok` Configuration][configuration] that allows users to define and simulate metrics to Node(s).

A Metrics resource has the following fields:

-->
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
    labels:
    - name: <string>
      value: <string>
    value: <string>   # for counter and gauge
    buckets:          # for histogram
    - le: <float64>
      value: <string>
      hidden: <bool>
      dimension: <string>
```


To simulate a metric, you can set the `metrics` field in the spec section of a Metrics resource.

The `path` field is a restful service path.
The `metrics` field is a list of metric configurations. Each metric configuration has the following fields:

[configuration]: {{< relref "/docs/user/configuration" >}}
[Metrics API]: {{< relref "/docs/generated/apis" >}}#kwok.x-k8s.io/v1alpha1.Metrics
