---
title: Simulation Testing
---

# Simulation Testing with KWOK

{{< hint "info" >}}

This document walks you through the technical outcome of using KWOK for simulation testing.

{{< /hint >}}

This article describes the technical outcome of using KWOK for simulation testing in your environment.
It goes over the key simulation features of using KWOK, and then provides a step-by-step guide to creating
various scheduling scenarios using KWOK.

## Key Simulation Features of KWOK

1. **Lifecycle configuration:** The pod, node, and node lease can be simulated.

    - **Pod:** Status information like Pending, ContainerCreating, Running, and Terminating can be simulated.
    - **Node:** Status information like NodeReady, and Ready states can be simulated.

2. **Kubelet server**

    - **Metrics:** Using the [Kubernetes Metric Server](https://kwok.sigs.k8s.io/docs/examples/metrics-server/), or [Prometheus](https://kwok.sigs.k8s.io/docs/examples/prometheus/), metrics can be simulated in KWOK.
    - **Exec:** Commands can be simulated to be [executed](https://kwok.sigs.k8s.io/docs/user/exec-configuration/) inside a container running on a single fake pod or multiple pods.
    - **Log:** Container [logs](https://kwok.sigs.k8s.io/docs/user/logs-configuration/) can be simulated in a single pod or multiple pods.
    - **Attach:** The input/output streams from the main process running in a container can be [attached](https://kwok.sigs.k8s.io/docs/user/attach-configuration/#attach-configuration) to KWOK.
    This can be simulated in a single pod or multiple pods.
    - **PortForward:** [Port forwarding](https://kwok.sigs.k8s.io/docs/user/port-forward-configuration/#clusterportforward), which establishes a direct connection to a container port, can be simulated
    for a single pod or multiple pods.

## Technical Outcome

### Scheduler Simulation

KWOK can be used to create fake nodes and pods in a simulated cluster.
The cluster can be configured with scheduling policies that meet your scheduler's requirements.
These scenarios below will be used to describe this:

- [Scenario 1: Scheduling pods with resource requests and limits](/docs/technical-outcomes/scheduling/requests-and-limits)
- [Scenario 2: Scheduling a pod to a particular node with node-affinity](/docs/technical-outcomes/scheduling/node-affinity)
- [Scenario 3: Scheduling pods with taints and tolerations](/docs/technical-outcomes/scheduling/taints-and-tolerations)
- [Scenario 4: Scheduling pods with a limit range](/docs/technical-outcomes/scheduling/limit-range)
- [Scenario 5: Scheduling pods using pod priority and preemption](/docs/technical-outcomes/scheduling/pod-priority-and-preemption)
- [Scenario 6: Scheduling pods using pod topology spread constraints](/docs/technical-outcomes/scheduling/pod-topology-spread-constraint)

Other scheduling scenarios can also be simulated using KWOK.
