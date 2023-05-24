# `kwokctl` Authorization

{{< hint "info" >}}

This document walks you known about kubernetes authorization in kwokctl.

{{< /hint >}}

## What is Authorization

Kubernetes provides a rich authorization plugin interface to support various authorization modules,
such as ABAC, RBAC, Webhook and Node.

## Enable Authorization

Authorization is enabled by default. but prior to v0.3.0, that is disabled by default (excluding kind).

Use `--kube-authorization=true` or `--kube-authorization=false` to enable or disable authorization when creating a cluster.
