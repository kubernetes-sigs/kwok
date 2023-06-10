---
title: "Admission"
---

# `kwokctl` Admission

{{< hint "info" >}}

This document walks you known about kubernetes admission in kwokctl.

{{< /hint >}}

## What is Admission

Kubernetes provides a mechanism called admission controller to intercept requests to the Kubernetes API server prior to persistence of the object, but after the request is authenticated and authorized.
There are two special controllers: `MutatingAdmissionWebhook` and `ValidatingAdmissionWebhook`.
These two controllers call out to a webhook service to do some processing.

## Enable Admission

Admission is enabled by default. but prior to v0.3.0, that is disabled by default (excluding kind).

Use `--kube-admission=true` or `--kube-admission=false` to enable or disable admission when creating a cluster.

If you are creating a cluster with kube version < `1.21`, then [authorization] also needs to be enabled.

## Webhook Configuration

### In Binary

All components run using local binary, you need to use `127.0.0.1` to access the webhook service on the host.

### In Docker/Podman/Nerdctl

All components run in containers, you need to use the service name of the network to access the webhook service on the container.

### In Kind

All components run in kind cluster, just like general kubernetes cluster.

[authorization]: {{< relref "/docs/user/kwokctl-authorization" >}}
