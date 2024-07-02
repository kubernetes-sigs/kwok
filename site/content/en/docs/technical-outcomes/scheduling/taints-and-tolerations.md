---
title: "Scenario 3"
---

# Scenario 3: Scheduling Pods with taints and tolerations

A taint and toleration can be defined in KWOK. It’s handy in these situations:

1. You have KWOK installed in your real cluster (KIND, K3D, etc.), and you want some or all of your pods to be scheduled to the KWOK nodes to test for scalability.
2. You want to simulate taint and toleration use cases within a KWOK cluster.

Let's look at **point 2** for now.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="../simulation/scenario-3/README.svg">

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

## Deploy node

- Node: 1 worker node in the cluster

{{< expand "../simulation/scenario-3/node.yaml" >}}

{{< code-sample file="../simulation/scenario-3/node.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f node.yaml
```

## Taint node

```bash
kubectl taint nodes node kwok.x-k8s.io/node=fake:NoSchedule
```

## Deploy a pod without toleration and observe

{{< expand "../simulation/scenario-3/no-toleration-pod.yaml" >}}

{{< code-sample file="../simulation/scenario-3/no-toleration-pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f no-toleration-pod.yaml
kubectl get pod

NAME                READY   STATUS    RESTARTS   AGE
no-toleration-pod   0/1     Pending   0          4s
```

The pod is stuck in a pending state.

## Deploy a pod with toleration and observe

{{< expand "../simulation/scenario-3/with-toleration-pod.yaml" >}}

{{< code-sample file="../simulation/scenario-3/with-toleration-pod.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f with-toleration-pod.yaml

kubectl get pod

NAME                    READY   STATUS    RESTARTS   AGE
no-toleration-pod       0/1     Pending   0          20s
with-toleration-pod     1/1     Running   0          2s
```

Only the pod with toleration is in a running state.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on [taints and tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) policy.
