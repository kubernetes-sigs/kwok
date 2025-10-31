---
title: Inject Failure Demo
aliases:
- /user/examples/injectfailure/
---

# Inject Failure Demo

{{< hint "info" >}}

This is a demo that introduces how to inject fault to initContainer in an existing minikube cluster.

{{< /hint >}}

<img width="700px" src="init-container-error-injection.svg">

{{< expand "Demo Detail Steps" >}}

{{< code-sample file="init-container-error-injection.demo" language="bash" >}}

{{< /expand >}}

{{< expand "virtual-gpu-node.yaml" >}}

{{< code-sample file="virtual-gpu-node.yaml" >}}

{{< /expand >}}

{{< expand "failed-pod.yaml" >}}

{{< code-sample file="failed-pod.yaml" >}}

{{< /expand >}}
