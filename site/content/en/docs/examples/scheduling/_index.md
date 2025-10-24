---
title: Scheduling
aliases:
- /docs/technical-outcomes/scheduling/
---

# Scheduling Testing with KWOK

{{< hint "info" >}}

This document walks you through the technical outcome of using KWOK for scheduler tests.

{{< /hint >}}

KWOK can be used to create fake nodes and pods in a simulated cluster.
The cluster can be configured with scheduling policies that meet your scheduler's requirements.
The scenarios below can be used to describe this:

- [Scheduling pods with resource requests and limits](/docs/technical-outcomes/scheduling/requests-and-limits)
- [Scheduling a pod to a particular node with node-affinity](/docs/technical-outcomes/scheduling/node-affinity)
- [Scheduling pods with taints and tolerations](/docs/technical-outcomes/scheduling/taints-and-tolerations)
- [Scheduling pods with a limit range](/docs/technical-outcomes/scheduling/limit-range)
- [Scheduling pods using pod priority and preemption](/docs/technical-outcomes/scheduling/pod-priority-and-preemption)
- [Scheduling pods using pod topology spread constraints](/docs/technical-outcomes/scheduling/pod-topology-spread-constraint)

Other scheduling scenarios can also be simulated using KWOK.
