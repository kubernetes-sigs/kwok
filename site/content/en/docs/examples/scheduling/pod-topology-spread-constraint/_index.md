---
title: "Scheduling pods using pod topology spread constraints"
aliases:
- /docs/technical-outcomes/scheduling/pod-topology-spread-constraint
---

# Scheduling pods using pod topology spread constraints

Pod topology spread constraints scheduling policies can be applied in a KWOK cluster.

For this particular scenario, the cluster has 4 nodes that span 2 regions.
A pod topology spread constraint policy will be applied, so the pods are evenly scheduled across each region.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="pod-topology-spread-constraint.svg">

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
kwokctl scale node --replicas 4
```

## Label nodes

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

```bash
kubectl label nodes node-000000 topology.kubernetes.io/region=us-west-1
kubectl label nodes node-000001 topology.kubernetes.io/region=us-west-1
kubectl label nodes node-000002 topology.kubernetes.io/region=us-west-2
kubectl label nodes node-000003 topology.kubernetes.io/region=us-west-2
```

## Create deployment

{{< expand "deployment.yaml" >}}

{{< code-sample file="deployment.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f deployment.yaml
```

The deployment creates 4 replicas, with its `maxSkew` set to 1.

## Observe the topology spread

```bash
kubectl get pod -o wide

NAME                        READY   STATUS    RESTARTS   AGE   IP         NODE          NOMINATED NODE   READINESS GATES
fake-app-685c7b9cc7-2n8lv   1/1     Running   0          5s    10.0.2.2   node-000002   <none>           <none>
fake-app-685c7b9cc7-2s2gf   1/1     Running   0          5s    10.0.1.1   node-000001   <none>           <none>
fake-app-685c7b9cc7-7zvm9   1/1     Running   0          5s    10.0.1.2   node-000001   <none>           <none>
fake-app-685c7b9cc7-fhntg   1/1     Running   0          5s    10.0.2.1   node-000002   <none>           <none>
```

All pods are evenly distributed among each node across the regions.

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate a scheduling
scenario based on setting a [pod topology spread constraints](https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/) policy.
