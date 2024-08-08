---
title: "Multi-Cluster Control Plane Performance Evaluation"
---

# Multi-cluster control plane performance evaluation

While KWOK is commonly used to test the performance of a [single Kubernetes cluster](../single-cluster/_index.md),
KWOK also supports multi-cluster environments, enabling users to evaluate performance across multiple clusters.
This can be particularly useful for scenarios involving complex orchestrators or multi-cloud strategies.

## Simulating Multi-Cluster Environments Using KWOK

Users can spin up multiple clusters using KWOK or they can use an existing non-KWOK cluster.

Below is an example of a multi-cluster created using KWOK.

```bash
for ((i=1; i<=2; i ++)); do
    kwokctl create cluster --name member$i
done;
```

Once the clusters are created, you can list the contexts to verify the setup:

```bash
kubectl config get-contexts 
```

The expected output will display the contexts for the KWOK-managed clusters:

```bash
CURRENT   NAME           CLUSTER        AUTHINFO       NAMESPACE
          kwok-member1   kwok-member1   kwok-member1   
*         kwok-member2   kwok-member2   kwok-member2   
```

In this example, two clusters named `kwok-member1` and `kwok-member2` are created.
You can switch between these clusters or manage them simultaneously as part of your multi-cluster testing strategy.

Users can also create multiple clusters using Kind, and then install the KWOK custom resource definition,
so fake nodes and pods can be deployed into it for performance evaluation.

## Using KWOK for performance evaluation

[Karmada](https://karmada.io), a multi-cloud, multi-cluster orchestrator, uses a [previous version of KWOK](https://github.com/wzshiming/fake-kubelet) to test its control plane.

Below are a few parts of its components that were tested using KWOK.
1. **Karmada-apiserver:**
    - **Purpose:** It exposes and handles Karmada and Kubernetes API requests.
    - **Test:** Evaluate how long it takes the API server to process requests that
    change the state (create, update, or delete) of custom resources (work objects), under high load.
2. **Karmada controller manager:**
    - **Purpose:** Runs several controllers that watch [custom resources (work)](https://karmada.io/docs/reference/karmada-api/work-resources/work-v1alpha1#workspec) created by Karmada.
    It also communicates with the API server of member clusters to create Kubernetes resources specified in the custom resource.
    - **Test:** Based on API requests, monitor to ensure that the Karmada Controller Manager can efficiently handle resource propagation requests and maintain low latency when distributing resources across member clusters based on defined policies.

## Reference

- [Performance Test Setup for Karmada](https://karmada.io/docs/developers/performance-test-setup-for-karmada/)
