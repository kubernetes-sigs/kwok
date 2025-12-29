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
kwok-node-0   Ready    agent   4s    kwok-v0.7.0
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
  name: kwok-resource-flavor
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
    - name: kwok-resource-flavor
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
NAME                                             COHORT        PENDING WORKLOADS
clusterqueue.kueue.x-k8s.io/kwok-cluster-queue   kwok-cohort   0

NAME                                         CLUSTERQUEUE         PENDING WORKLOADS   ADMITTED WORKLOADS
localqueue.kueue.x-k8s.io/kwok-local-queue   kwok-cluster-queue   0                   0

NAME                                                 AGE
resourceflavor.kueue.x-k8s.io/kwok-resource-flavor   27s

NAME                                AGE
cohort.kueue.x-k8s.io/kwok-cohort   45s

NAME                                    AGE
topology.kueue.x-k8s.io/kwok-topology   39s
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
kubectl get workloads
NAME                   QUEUE              RESERVED IN          ADMITTED   FINISHED   AGE
job-sample-job-d7b38   kwok-local-queue   kwok-cluster-queue   True       True       5s
```

```bash
kubectl describe workload job-sample-job-d7b38
Name:         job-sample-job-d7b38
Namespace:    default
Labels:       kueue.x-k8s.io/job-uid=22cd157f-9ddd-43bb-9e88-223ff22c146c
Annotations:  <none>
API Version:  kueue.x-k8s.io/v1beta1
Kind:         Workload
Metadata:
  Creation Timestamp:  2025-12-29T19:24:23Z
  Generation:          1
  Owner References:
    API Version:           batch/v1
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  Job
    Name:                  sample-job
    UID:                   22cd157f-9ddd-43bb-9e88-223ff22c146c
  Resource Version:        906
  UID:                     2d40bf89-86a4-4790-ba30-939385be714b
Spec:
  Active:  true
  Pod Sets:
    Count:  1
    Name:   main
    Template:
      Metadata:
      Spec:
        Containers:
          Args:
            30s
          Image:              gcr.io/k8s-staging-perf-tests/sleep:v0.1.0
          Image Pull Policy:  IfNotPresent
          Name:               dummy-job
          Resources:
            Requests:
              Cpu:                     1
              Memory:                  200Mi
          Termination Message Path:    /dev/termination-log
          Termination Message Policy:  File
        Dns Policy:                    ClusterFirst
        Node Selector:
          Type:          kwok
        Restart Policy:  Never
        Scheduler Name:  default-scheduler
        Security Context:
        Termination Grace Period Seconds:  30
    Topology Request:
      Pod Index Label:    batch.kubernetes.io/job-completion-index
  Priority:               0
  Priority Class Source:  
  Queue Name:             kwok-local-queue
Status:
  Admission:
    Cluster Queue:  kwok-cluster-queue
    Pod Set Assignments:
      Count:  1
      Flavors:
        Cpu:     kwok-resource-flavor
        Memory:  kwok-resource-flavor
      Name:      main
      Resource Usage:
        Cpu:     1
        Memory:  200Mi
      Topology Assignment:
        Domains:
          Count:  1
          Values:
            kwok
        Levels:
          type
  Conditions:
    Last Transition Time:  2025-12-29T19:24:23Z
    Message:               Quota reserved in ClusterQueue kwok-cluster-queue
    Observed Generation:   1
    Reason:                QuotaReserved
    Status:                True
    Type:                  QuotaReserved
    Last Transition Time:  2025-12-29T19:24:23Z
    Message:               The workload is admitted
    Observed Generation:   1
    Reason:                Admitted
    Status:                True
    Type:                  Admitted
    Last Transition Time:  2025-12-29T19:24:25Z
    Message:               Reached expected number of succeeded pods
    Observed Generation:   1
    Reason:                Succeeded
    Status:                True
    Type:                  Finished
Events:
  Type    Reason         Age   From             Message
  ----    ------         ----  ----             -------
  Normal  QuotaReserved  15s   kueue-admission  Quota reserved in ClusterQueue kwok-cluster-queue, wait time since queued was 1s
  Normal  Admitted       15s   kueue-admission  Admitted by ClusterQueue kwok-cluster-queue, wait time since reservation was 0s
...
```

Verify the job is admitted, executed and completed
```bash
kubectl get jobs
NAME         STATUS     COMPLETIONS   DURATION   AGE
sample-job   Complete   1/1           2s         37s
```
