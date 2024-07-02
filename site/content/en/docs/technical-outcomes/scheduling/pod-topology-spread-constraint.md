---
title: "Scenario 6"
---

# Scenario 6: Scheduling pods using pod topology spread constraints

Pod topology spread constraints scheduling policies can be applied in a KWOK cluster.

For this particular scenario, the cluster has 4 nodes that span 2 regions.
A pod topology spread constraint policy will be applied, so the pods are evenly scheduled across each region.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="../simulation/scenario-6/README.svg">

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

Below are the node resource specifications:

- Node: 4 worker nodes in the cluster
  - Node 1: 
    - Key = topology.kubernetes.io/region
    - Value = us-west-1
  - Node 2: 
    - Key = topology.kubernetes.io/region
    - Value = us-west-1
  - Node 3:
    - Key = topology.kubernetes.io/region
    - Value = us-west-2
  - Node 4:
    - Key = topology.kubernetes.io/region
    - Value = us-west-2

{{< expand "../simulation/scenario-6/node-in-us-west-1.yaml" >}}

{{< code-sample file="../simulation/scenario-6/node-in-us-west-1.yaml" >}}

{{< /expand >}}

{{< expand "../simulation/scenario-6/node-in-us-west-2.yaml" >}}

{{< code-sample file="../simulation/scenario-6/node-in-us-west-2.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f node-in-us-west-1.yaml
kubectl apply -f node-in-us-west-2.yaml
```

## Create deployment

{{< expand "../simulation/scenario-6/deployment.yaml" >}}

{{< code-sample file="../simulation/scenario-6/deployment.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f deployment.yaml
```

The deployment creates 4 replicas, with its `maxSkew` set to 1.

## Observe the topology spread

```bash
kubectl get pod -o wide
NAME                        READY   STATUS    RESTARTS   AGE   IP         NODE     NOMINATED NODE   READINESS GATES
fake-app-685c7b9cc7-6rdgn   1/1     Running   0          13s   10.0.0.4   node-2   <none>           <none>
fake-app-685c7b9cc7-gcgsq   1/1     Running   0          13s   10.0.0.2   node-4   <none>           <none>
fake-app-685c7b9cc7-ssjq9   1/1     Running   0          13s   10.0.0.3   node-1   <none>           <none>
fake-app-685c7b9cc7-tqxrs   1/1     Running   0          13s   10.0.0.1   node-3   <none>           <none>
```

All pods are evenly distributed among each node across the regions.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on setting a [pod topology spread constraints](https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/) policy.
