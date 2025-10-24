---
title: "Vertical Pod Autoscaling"
aliases:
- /docs/technical-outcomes/scalability/scale-using-vpa/
---

# Vertical pod autoscaling

Using KWOK, we can deploy a metric server to help us observe and trigger our Vertical pod autoscaler.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="scale-using-vpa.svg">

## Prerequisites

- KWOK must be installed on the machine. See [installation](https://kwok.sigs.k8s.io/docs/user/installation/).
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)

## Get metrics usage

```bash
wget https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/metrics-usage.yaml
```

## Create cluster

```bash
kwokctl create cluster --enable-metrics-server --config ./metrics-usage.yaml --runtime binary
```

The arguments `--enable-metrics-server --config ./metrics-usage.yaml` provides the ability to
use a metric server in the cluster.

The `--runtime binary` argument means that the cluster will run applications directly on the host system instead of using containerized environments. To view the running components of the cluster, for example, the metrics-server, you can do a `ps aux | grep -i metrics-server`.

## Create a node

```bash
kwokctl scale node --replicas 1 --param '.allocatable.cpu="4000m"'
```

## Create deployment

{{< expand "deployment.yaml" >}}

{{< code-sample file="deployment.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f deployment.yaml
```

- Deployment specification:
  - CPU Request: 100m

## Set up VPA

```bash
kubectl apply -f ./autoscaler/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml
kubectl apply -f ./autoscaler/vertical-pod-autoscaler/deploy/vpa-rbac.yaml
```

These commands sets up the Vertical Pod Autoscaler (VPA)
in the Kubernetes cluster by applying the necessary Custom Resource Definitions (CRDs)
and Role-Based Access Control (RBAC) configurations.

The [Kubernetes autoscaler repository](https://github.com/kubernetes/autoscaler) needs to be cloned before executing the above command.

## Deploy the three components of the Vertical Pod Autoscaler

Note that the command references the cloned GitHub repository. So, locate the folder where the repository
is saved before executing each command.

The admission-controller:

```bash
{ cd autoscaler/vertical-pod-autoscaler/ && NAMESPACE=kube-system go run ./pkg/admission-controller --kubeconfig ~/.kube/config --client-ca-file ~/.kwok/clusters/kwok/pki/ca.crt --tls-cert-file  ~/.kwok/clusters/kwok/pki/admin.crt --tls-private-key  ~/.kwok/clusters/kwok/pki/admin.key --webhook-address https://127.0.0.1 --webhook-port 8080 --register-by-url --port 8080 ;} &
```

The recommender:

```bash
{ cd autoscaler/vertical-pod-autoscaler/ && NAMESPACE=kube-system go run ./pkg/recommender --kubeconfig ~/.kube/config ;} &
```

The updater:

```bash
{ cd autoscaler/vertical-pod-autoscaler/ && NAMESPACE=kube-system go run ./pkg/updater --kubeconfig ~/.kube/config ;} &
```

You should see several outputs similar to the below:

```bash
I0724 18:13:32.418318   15823 api.go:94] Initial VPA synced successfully
I0724 18:13:32.474487   15829 controller_fetcher.go:141] Initial sync of DaemonSet completed
I0724 18:13:32.513184   15836 fetcher.go:99] Initial sync of ReplicaSet completed
I0724 18:13:32.519144   15823 fetcher.go:99] Initial sync of DaemonSet completed
I0724 18:13:32.575933   15829 controller_fetcher.go:141] Initial sync of Deployment completed
I0724 18:13:32.613517   15836 fetcher.go:99] Initial sync of StatefulSet completed
```

## Create the VPA

{{< expand "vpa.yaml" >}}

{{< code-sample file="vpa.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f vpa.yaml
```

This VPA uses an `Auto` mode, meaning that the autoscaler analyzes the existing pods,
recommends a suitable CPU and memory request, and then applies the configuration.

## Wait until the recommendation is provided

```bash
kubectl wait --for=condition=RecommendationProvided=true vpa/fake-vpa --timeout=120s
```

This command waits for the Vertical Pod Autoscaler (VPA) resource named `fake-vpa` to reach the condition where a recommendation has been provided.

You should get the below output.

```bash
verticalpodautoscaler.autoscaling.k8s.io/fake-vpa condition met
```

## Check recommended VPA values

```bash
kubectl describe vpa fake-vpa | awk '/Recommendation:/,/Events:/' | sed '$d'

Recommendation:
Container Recommendations:
  Container Name:  fake-app
  Lower Bound:
    Cpu:     25m
    Memory:  262144k
  Target:
    Cpu:     25m
    Memory:  262144k
  Uncapped Target:
    Cpu:     25m
    Memory:  262144k
  Upper Bound:
    Cpu:     3179m
    Memory:  3323500k
```

## Wait 60s

```bash
sleep 60
```

During this period, the deployment terminates existing pods and recreates new pods with the recommended values from the VPA.

## See the new applied CPU limits

```bash
kubectl get pod -o yaml | grep -i cpu:

cpu: 25m
cpu: 25m
```

## Delete the cluster

```bash
kwokctl delete cluster
```

After deleting the cluster, you should close the terminal, or else the VPA 
components deployed may keep trying to communicate
to the API server, resulting in several error outputs.

## Conclusion

This example demonstrates how to use KWOK to simulate [vertical
pod autoscaling](https://github.com/kubernetes-sigs/cluster-proportional-vertical-autoscaler).

Other VPA modes can also be simulated.
