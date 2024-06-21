---
title: "Limit Range"
---

# Scheduling pods with a limit range

A limit range schedule policy can be used in a KWOK cluster.

<img width="700px" src="limit-range.svg">

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

## Create nodes

```bash
kwokctl scale node --replicas 1
```

## Create a resource limit

{{< expand "limit-range.yaml" >}}

{{< code-sample file="limit-range.yaml" >}}

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

- Pod specification:
  - CPU Request: 700m

{{< expand "pod-beyond-limit.yaml" >}}

{{< code-sample file="pod-beyond-limit.yaml" >}}

{{< /expand >}}

```bash
kubectl create -f pod-beyond-limit.yaml
```

Notice the error `Invalid value: "700m": must be less than or equal to cpu limit of 500m`

## Deploy a pod within the resource limit

- Pod specification:
  - CPU Request: 400m

{{< expand "pod-within-limit.yaml" >}}

{{< code-sample file="pod-within-limit.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f pod-within-limit.yaml
```

## Confirm that the pod is running

```bash
kubectl get pod

NAME                 READY   STATUS    RESTARTS   AGE
pod-within-limit   1/1     Running   0          10s
```

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on setting a [limit range](https://kubernetes.io/docs/concepts/policy/limit-range/) policy.
