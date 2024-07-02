---
title: "Scenario 2"
---

# Scenario 2: Scheduling a pod to a particular node with node-affinity

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="../simulation/scenario-2/README.svg">

## Prerequisites

- KWOK must be installed on the machine. See [installation](https://kwok.sigs.k8s.io/docs/user/installation/).
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)

## Create cluster

```bash
kwokctl create cluster
```

## View clusters

- This ensures that the cluster was created successfully.

```bash
kwokctl get clusters
```

## Deploy nodes

Below are the node resource specifications:

- Nodes: 2 worker nodes in the cluster

```bash
kubectl apply -f node.yaml
```

{{< expand "../simulation/scenario-2/node.yaml" >}}

{{< code-sample file="../simulation/scenario-2/node.yaml" >}}

{{< /expand >}}

Label node-1

```bash
kubectl label node node-1 region=us-west-2
```

## Pod to be scheduled

The pod has a node affinity configured with a key value of “region=eu-west-2”. This matches the label on node-1.

## Scheduling process

### Step 1: Deploy pod

{{< expand "../simulation/scenario-2/pod.yaml" >}}

{{< code-sample file="../simulation/scenario-2/pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f pod.yaml
```

### Step 2: View the node the pod is scheduled to.

```bash
kubectl get pod -o wide
```

- Pod 1 is scheduled to Node 1.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling 
scenario based on [node affinity](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes-using-node-affinity/) policy.
