# Introduction

{{< hint "info" >}}

This document will introduce the design of Kwok.

{{< /hint >}}

## What's the kubemark?

[kubemark] is a kubelet that does not actually run a container.

## What's the kind?

[kind] is run Kubernetes in Docker that is a real cluster.

## User Stories

### Scheduler

As a scheduler developer, I want to test the scheduler with a large number of Nodes and Pods,

### CRD Controller 

As a CRD controller developer, I want to test the controller without fake clients.

### Control Plane Performance

As a control plane performance tester, I want to test the performance of the control plane at a low cost.

## What do we want?

- Low cost simulation any cluster.
- Use like a real cluster.
- Customizable emulation.
- Runs in any environment.
- Fast startup.
- Easy to use.

## Predecessor

This project was originally a migration of [fake-kubelet] and [fake-k8s] projects.

[fake-kubelet]: https://github.com/wzshiming/fake-kubelet
[fake-k8s]: https://github.com/wzshiming/fake-k8s
[kind]: https://github.com/kubernetes-sigs/kind
[kubemark]: https://github.com/kubernetes/kubernetes/tree/master/test/kubemark
