---
title: Introduction
---

# Introduction

{{< hint "info" >}}

This document will introduce the design of KWOK.

{{< /hint >}}

## User Stories

### Scheduler

As a scheduler developer, I want to test the scheduler with a large number of Nodes and Pods.

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

## Next Steps

Learn more about the [architecture] of KWOK.

[fake-kubelet]: https://github.com/wzshiming/fake-kubelet
[fake-k8s]: https://github.com/wzshiming/fake-k8s
[architecture]: {{< relref "/docs/design/architecture" >}}
