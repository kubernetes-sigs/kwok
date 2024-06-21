---
title: "Node Affinity"
---

# Scheduling a pod to a particular node using node-affinity

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="node-affinity.svg">

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

## Create nodes

```bash
kwokctl scale node --replicas 2
```

Label node-000000

```bash
kubectl label node node-000000 region=us-west-2
```

## Scheduling process

### Step 1: Deploy pod

The pod has a node affinity configured with a key value of `region=eu-west-2`. This matches the label on node-000000.
{{< expand "pod.yaml" >}}

{{< code-sample file="pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f pod.yaml
```

### Step 2: View the node the pod is scheduled to.

```bash
kubectl get pod -o wide
```

- Pod 1 is scheduled to node-000000.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling 
scenario based on [node affinity](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes-using-node-affinity/) policy.
