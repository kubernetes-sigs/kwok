---
title: "Scheduling pods using pod priority and preemption"
---

# Scheduling pods using pod priority and preemption

Pod priority and preemption scheduling policies can be applied in a KWOK cluster.
For this particular scenario, the cluster will be limited to a particular resource range, then
pod priority and preemption policy policies will be used to evict low-priority pods.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="pod-priority-and-preemption.svg">

## Prerequisites

- KWOK must be installed on the machine. See [installation](https://kwok.sigs.k8s.io/docs/user/installation/).
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)

## Create cluster

```bash
kwokctl create cluster
```

## View clusters

This ensures that the cluster was created successfully.

```bash
kwokctl get clusters
```

## Create a node

```bash
kwokctl scale node --replicas 1 --param '.allocatable.cpu="4000m"'
```

## Create priority classes (low and high)

{{< expand "priority-classes.yaml" >}}

{{< code-sample file="priority-classes.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f priority-classes.yaml
```

## Deploy a low-priority Pod

This allows any pod that matches this priority class to consume **3 CPUs** off the node.

{{< expand "low-priority-pod.yaml" >}}

{{< code-sample file="low-priority-pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f low-priority-pod.yaml
```

## Deploy a high-priority pod

This allows any pod that matches this priority class to consume **3 CPUs** off the node.
Since the node has a maximum of **4 CPUs**, pods 
with a low-priority class that consumes over **1 CPU** will be preempted.

{{< expand "high-priority-pod.yaml" >}}

{{< code-sample file="high-priority-pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f high-priority-pod.yaml
```

## Observe Preemption

Now, observe the preemption process. 
The higher priority pod will preempt the lower priority pod due to limited resources.

```bash
kubectl get pods

NAME                READY   STATUS    RESTARTS   AGE
high-priority-pod   1/1     Running   0          5s
```

You should see that the low-priority pod has been
terminated and the high-priority pod has been scheduled.

## See more details about the preemption event

```bash
kubectl describe pod high-priority-pod | awk '/Events:/,/pod to node/'

Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  11m   default-scheduler  0/1 nodes are available: 1 Insufficient cpu.
  Normal   Scheduled         11m   default-scheduler  Successfully assigned default/high-priority-pod to node-000000
```

The event log indicates that the high-priority pod preempted another pod due to its higher priority.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on setting a [priority and preemption](https://kubernetes.io/docs/concepts/scheduling-eviction/) policy.
