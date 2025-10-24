---
title: "Single Cluster Control Plane Performance Evaluation"
aliases:
- /docs/technical-outcomes/performance/control-plane/single-cluster/
---

# Single cluster control plane performance evaluation

The Kubernetes control plane is crucial for managing the state and health of a Kubernetes cluster. It consists of several key components:

* **API server:** Handles requests to create, list, and delete resources.
* **etcd:** Stores cluster data and state.
* **Scheduler:** Places pods on appropriate nodes.
* **Controller manager:** Ensures resources match their desired state.

A malfunctioning Kubernetes control plane or a custom control plane (which extends the Kubernetes API) can severely 
impact the cluster’s ability to deploy, scale, or manage applications, causing downtime. Therefore, evaluating the 
performance of the control plane is essential for maintaining high availability and reliability.

## Using KWOK for performance evaluation

Since running large clusters can be expensive, KWOK helps simulate and test the control plane’s performance in a single cluster.
For example, [Kubevirt](https://github.com/kubevirt/kubevirt), a virtualization API platform that allows users to
run bare-metal Virtual Machine Instances (VMI) in a Kubernetes cluster, uses KWOK.

### KWOK tests for the control plane

KubeVirt tests the following control plane components using KWOK:

1. **Kubernetes control plane:**
    -  **API server:**
        * **Purpose:** Handles requests for creating, listing, and deleting resources.
        * **Test:** Simulates the creation of 100 KWOK nodes and 1,000 VMIs to check if the API server can handle a high number of requests and manage resources effectively.
    -  **Scheduler:**
        * **Purpose:** Places a pod’s VMI on appropriate nodes based on resource availability and scheduling policies.
        * **Test:** Creates many fake VMIs to see how well the scheduler distributes them across nodes.
    -  **Controller manager:**
        * **Purpose:** Oversees various controllers that maintain the desired state of a cluster. For example, the [Node controller](https://kubernetes.io/docs/concepts/overview/components/#kube-controller-manager)
        ensures nodes are healthy and available, while the KubeVirt controller manages the lifecycle of VMIs.
        * **Test:** Evaluates how well the controller manager handles tasks like checking node health and managing VMIs under heavy load.
2. **KubeVirt control plane:**
    * **Purpose:** Comprises components responsible for managing VMIs. Components such as the [Virt-Launcher, Virt-Controller, and Virt-Handler](https://kubernetes.io/blog/2018/05/22/getting-to-know-kubevirt/)
    * **Test:** Check if KubeVirt components work correctly with the Kubernetes control plane to ensure VMIs are created, scheduled, and started correctly.

### Lifecycle evaluation of the KubeVirt control plane

KubeVirt evaluates the following lifecycle aspects of its control plane:

* **Creation:** Tests the ability to create many Virtual Machine Instances (VMIs) efficiently. This involves API 
interactions for VMI creation and scheduling.

* **Scheduling:** Assesses how well VMIs are assigned to nodes based on available resources and policies.

* **Management:** Evaluates how well the control plane handles errors, allocates resources, and manages VMIs' state changes
(e.g., from creation to running).

* **Monitoring:** Checks the monitoring of VMIs to track their health and status, ensuring that the control plane can manage 
and update their state as required.

* **Cleanup:** Evaluates how effectively VMIs and their resources are deleted and cleaned up.

This assessment ensures the KubeVirt control plane can handle the full lifecycle of VMIs under high-density conditions,
from creation to clean-up, while maintaining performance and reliability.

## Reference

Beyond Kubevirt, other projects uses KWOK in a similar way for performance testing.

- [How KubeVirt uses KWOK for performance testing](https://github.com/kubevirt/kubevirt/pull/12117)
- [How Karpenter uses KWOK for performance testing](https://github.com/kubernetes-sigs/karpenter/blob/main/test/suites/perf/scheduling_test.go)
- [How Kyverno uses KWOK for performance testing](https://github.com/kyverno/kyverno/tree/main/docs/perf-testing)
- [How Yunikorn uses KWOK for performance testing](https://github.com/apache/yunikorn-k8shim/blob/master/deployments/kwok-perf-test/kwok-setup.sh)
- [How Headlamp uses KWOK for performance testing](https://github.com/headlamp-k8s/headlamp/blob/main/docs/development/testing.md)
