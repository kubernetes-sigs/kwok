---
title: "Horizontal Pod Autoscaling"
---

# Horizontal pod autoscaling

Using KWOK, we can deploy a metric server to help us observe and trigger our HorizontalPodAutoscaler.

This image shows you what you should expect when testing this scenario.
You can follow the step-by-step guide after seeing this.

<img width="700px" src="scale-using-hpa.svg">

## Prerequisites

- KWOK must be installed on the machine. See [installation](https://kwok.sigs.k8s.io/docs/user/installation/).
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)

## Get metrics usage

```bash
wget https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/metrics-usage.yaml
```

## Create cluster

```bash
kwokctl create cluster --enable-metrics-server --config ./metrics-usage.yaml
```

The arguments `--enable-metrics-server --config ./metrics-usage.yaml` provides the ability to
use a metric server in the cluster. The metric server docker image will be downloaded and running as a docker container.

## Create a node

```bash
kwokctl scale node --replicas 1
```

## Create the pod

{{< expand "deployment.yaml" >}}

{{< code-sample file="deployment.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f deployment.yaml
```

- Deployment specification:
  - Current CPU usage: 200m
  - CPU Request: 1000m
  - HPA Utilization percentage = (Current CPU usage / CPU Request) * 100)
    - (200/1000) * 100 ≈ 20%

The current CPU usage was defined in the deployment manifest and passed via the below annotation.

```yaml
annotations:
    kwok.x-k8s.io/usage-cpu: 200m
```

## Wait for 50s

```bash
sleep 50
```

This gives the metric-server time to collect metrics of the pod.

## Confirm metric of pod

```bash
kubectl top pod

NAME                       CPU(cores)   MEMORY(bytes)   
fake-app-bdccf9b7f-phwtl   201m         1Mi  
```

## Deploy HPA

{{< expand "hpa.yaml" >}}

{{< code-sample file="hpa.yaml" >}}

{{< /expand >}}

```bash
kubectl apply -f hpa.yaml
```

- Core HorizontalPodAutoscaler specification:
  - MINPODS = 1
  - MAXPODS = 2
  - average CPU Utilization = 70%
  - scaleUp/scaleDown = 100% (meaning increase/decrease current pod by a 100%)

## Wait for 5s

```bash
sleep 5
```

## See more details about the HPA event before scaleUp

```bash
kubectl get hpa

NAME              REFERENCE             TARGETS        MINPODS   MAXPODS   REPLICAS   AGE
pod-auto-scaler   Deployment/fake-app   cpu: 20%/70%   1         2         1          9s
```

Only one replica in the deployment is running, and the threshold is at 20%
which is less than 70%.

## Let's increase the usage metrics of the pod

```bash
POD_NAME=$(kubectl top pods | awk 'NR>1 {print $1}')
kubectl patch pod $POD_NAME --type=json -p='[{"op":"add","path":"/metadata/annotations","value":{"kwok.x-k8s.io/usage-cpu":"800m"}}]'
```

This command extracts and lists the names of the pods.
Then it patches the current CPU usage of each pod's CPU to "800m". Here we have
just one pod, so it uses that alone.
This causes the CPU utilization of the pod to be over the threshold of 70%

## Wait for 30s

```bash
sleep 30
```

## See more details about the HPA event after scaleUp

```bash
kubectl describe hpa | awk '/Events:/ {found=1} found {print}' | tail -n 1

Normal  SuccessfulRescale  20s   horizontal-pod-autoscaler  New size: 2; reason: cpu resource utilization (percentage of request) above target
```

This command displays the event data for when the deployment was auto-scaled to a 100%
by the HPA.

## Get scaled up HPA CPU utilization metrics

```bash
kubectl get hpa


NAME              REFERENCE             TARGETS        MINPODS   MAXPODS   REPLICAS   AGE
pod-auto-scaler   Deployment/fake-app   cpu: 50%/70%   1         2         2          54s
```

We can see now that the CPU utilization of the deployment is less than the threshold.
This is because the replica of the deployment is now two.

## Let's reduce the usage metrics to the pods

```bash
POD_NAME=$(kubectl top pods | awk 'NR>1 {print $1}')
kubectl patch pod $POD_NAME --type=json -p='[{"op":"add","path":"/metadata/annotations","value":{"kwok.x-k8s.io/usage-cpu":"200m"}}]'
```

This command extracts and lists the names of the pods.
Then it patches the current CPU usage of each pod's CPU to "200m".
This causes the CPU utilization of the pod to be less than the threshold of 70%,
hence, causing the HPA to scaled down by a 100%.

## Wait for 30s

```bash
sleep 30
```

## See more details about the latest HPA event after scaleDown

```bash
kubectl describe hpa | awk '/Events:/ {found=1} found {print}' | tail -n 1

Normal  SuccessfulRescale  16s  horizontal-pod-autoscaler  New size: 1; reason: All metrics below target
```

## Wait for 10s

```bash
sleep 10
```

## Get scaled down HPA CPU utilization metrics

```bash
kubectl get hpa

NAME              REFERENCE             TARGETS        MINPODS   MAXPODS   REPLICAS   AGE
pod-auto-scaler   Deployment/fake-app   cpu: 20%/70%   1         2         1          2m23s
```

## Delete the cluster

```bash
kwokctl delete cluster
```

## Conclusion

This example demonstrates how to use KWOK to simulate [horizontal
pod autoscaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).
