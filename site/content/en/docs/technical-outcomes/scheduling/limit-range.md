---
title: "Scenario 4"
---

# Scenario 4: Scheduling pods with a limit range

A limit range schedule policy can be used in a KWOK cluster.

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

## Deploy node

- Node: 1 worker node in the cluster

{{< expand "../simulation/scenario-4/node.yaml" >}}

{{< code-sample file="../simulation/scenario-4/node.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f node.yaml
```

## Create a resource limit

{{< expand "../simulation/scenario-4/limit-range.yaml" >}}

{{< code-sample file="../simulation/scenario-4/limit-range.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f limit-range.yaml
```

## Confirm the limit has the required values

```bash
kubectl describe limitranges cpu-resource-constraint

Name:       cpu-resource-constraint
Namespace:  default
Type        Resource  Min   Max  Default Request  Default Limit  Max Limit/Request Ratio
----        --------  ---   ---  ---------------  -------------  -----------------------
Container   cpu       100m  1    500m             500m           -
```

## Deploy a pod above the resource limit

- Pod specification: 0.7 CPU

{{< expand "../simulation/scenario-4/pod-beyond-limit.yaml" >}}

{{< code-sample file="../simulation/scenario-4/pod-beyond-limit.yaml" >}}

{{< /expand >}}

```bash
kubectl create -f pod-beyond-limit.yaml
```

Notice the error `Invalid value: "700m": must be less than or equal to cpu limit of 500m`

## Deploy a pod within the resource limit

- Pod specification: 0.4 CPU

{{< expand "../simulation/scenario-4/pod-within-limit.yaml" >}}

{{< code-sample file="../simulation/scenario-4/pod-within-limit.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f pod-within-limit.yaml
```

## Confirm that the pod is running

```bash
kubectl get pod

NAME                 READY   STATUS    RESTARTS   AGE
pod-with-new-limit   1/1     Running   0          10s
```

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on setting a [limit range](https://kubernetes.io/docs/concepts/policy/limit-range/) policy.
