---
title: Performance Testing
---

# Performance Testing with KWOK

{{< hint "info" >}}

This document walks you through the technical outcome of using KWOK for performance tests.

{{< /hint >}}

In Kubernetes environments, performance testing is crucial for maintaining the health and security of the cluster.
However, running these tests on real nodes can be expensive and resource-intensive.
KWOK can be used to create fake nodes and pods in simulated KWOK clusters or in an existing non-KWOK cluster.
The scenarios below can be used to describe this:

- [Optimizing etcd Performance](/docs/technical-outcomes/performance/etcd)
- [Control Plane Performance Evaluation](/docs/technical-outcomes/performance/control-plane)

Other performance scenarios can also be simulated using KWOK.
