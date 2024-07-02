---
title: "Scenario 1"
---

# Scenario 1: Scheduling pods with resource requests and limits

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="../simulation/scenario-1/README.svg">

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
  - Node 1: 4 CPUs
  - Node 2: 2 CPUs

{{< expand "../simulation/scenario-1/node.yaml" >}}

{{< code-sample file="../simulation/scenario-1/node.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f node.yaml
```

## Pod resource usage specifications

- Pod 1: Requests 2.7 CPU. Limits 3 CPU.
- Pod 2: Requests 1.2 CPU. Limits 1.5 CPU.

## Scheduling process

### Step 1: Deploy Pod 1 and 2

{{< expand "../simulation/scenario-1/pod-1.yaml" >}}

{{< code-sample file="../simulation/scenario-1/pod-1.yaml" >}}

{{< /expand >}}

{{< expand "../simulation/scenario-1/pod-2.yaml" >}}

{{< code-sample file="../simulation/scenario-1/pod-2.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f pod-1.yaml
kubectl apply -f pod-2.yaml
```

### Step 2: View the node the pod is scheduled to

```bash
kubectl get pod -o wide

NAME    READY   STATUS    RESTARTS   AGE   IP         NODE     NOMINATED NODE   READINESS GATES
pod-1   1/1     Running   0          5s    10.0.0.1   node-1   <none>           <none>
pod-2   1/1     Running   0          5s    10.0.0.2   node-2   <none>           <none>
```

- Pod 1 is scheduled to Node 1.
- Pod 2 is scheduled to Node 2.

### Step 3: View node resource usage

```bash
kubectl describe node node-1 | awk '/Allocated resources:/,/ephemeral-storage/'
kubectl describe node node-2 | awk '/Allocated resources:/,/ephemeral-storage/'
```

This will provide detailed information about the node's resource capacity and usage.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling scenario based on [resource requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/) policy.

