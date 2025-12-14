---
title: "Kueue"
---

# Kueue

More information about Kueue can be found [here](https://kueue.sigs.k8s.io/).

## Set up Cluster

Create a new KWOK cluster using the kind runtime
``` bash
kwokctl create cluster --runtime kind
```

## Create Node

Create KWOK fake nodes (the example node already has type=kwok label)
``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

Verify that the nodes have the correct label using a label selector
```bash
kubectl get nodes -l type=kwok
```

## Deploy Kueue

Install Kueue controller-manager
```bash
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/v0.14.4/manifests.yaml
```

Patch the Kueue controller-manager deployment to run on the control plane node
```bash
kubectl patch deploy kueue-controller-manager -n kueue-system --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

Wait for Kueue to be ready
```bash
kubectl wait --for=condition=available --timeout=300s deployment/kueue-controller-manager -n kueue-system
```

## Create Kueue Resources

Create a Cohort for resource sharing between ClusterQueues
```bash
kubectl apply -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: Cohort
metadata:
  name: kwok-cohort
EOF
```

Create a Topology to define workload scheduling levels
```bash
kubectl apply -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: Topology
metadata:
  name: kwok-topology
spec:
  levels:
  - nodeLabel: type
EOF
```

Create a ResourceFlavor to define resource characteristics
```bash
kubectl apply -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: ResourceFlavor
metadata:
  name: kwok-resource-flavor-a-east-r1
spec:
  nodeLabels:
    type: kwok
  topologyName: kwok-topology
EOF
```

Create a ClusterQueue with quota management policies
```bash
kubectl apply -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: kwok-cluster-queue
spec:
  cohort: kwok-cohort
  flavorFungibility:
    whenCanBorrow: Borrow
    whenCanPreempt: TryNextFlavor
  namespaceSelector: {}
  preemption:
    borrowWithinCohort:
      policy: LowerPriority
    reclaimWithinCohort: Any
    withinClusterQueue: LowerPriority
  queueingStrategy: BestEffortFIFO
  resourceGroups:
  - coveredResources:
    - cpu
    - memory
    flavors:
    - name: kwok-resource-flavor-a-east-r1
      resources:
      - borrowingLimit: "390144"
        name: cpu
        nominalQuota: "1000"
      - borrowingLimit: 2568Ti
        name: memory
        nominalQuota: 1Ti
  stopPolicy: None
EOF
```

Create a LocalQueue in the default namespace
```bash
kubectl apply -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: LocalQueue
metadata:
  name: kwok-local-queue
  namespace: default
spec:
  clusterQueue: kwok-cluster-queue
EOF
```

Verify the Kueue resources are created
```bash
kubectl get clusterqueues,localqueues,resourceflavors,cohorts,topologies
```

## Test with a Sample Job

Create a test job that uses the Kueue local queue
```bash
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: sample-job
  namespace: default
  labels:
    kueue.x-k8s.io/queue-name: kwok-local-queue
spec:
  parallelism: 1
  completions: 1
  suspend: true
  template:
    spec:
      containers:
      - name: dummy-job
        image: gcr.io/k8s-staging-perf-tests/sleep:v0.1.0
        args: ["30s"]
        resources:
          requests:
            cpu: "1"
            memory: "200Mi"
      restartPolicy: Never
      nodeSelector:
        type: kwok
EOF
```

Check the workload status
```bash
kubectl get workloads -n default
```

Verify the job is admitted and running
```bash
kubectl get jobs
# Describe the workload (use the workload name from the get workloads output)
kubectl describe workload -n default
```
