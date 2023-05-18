# `kwokctl` Authorization

{{< hint "info" >}}

This document walks you known about kubernetes authorization in kwokctl.

{{< /hint >}}

## What is Authorization

Kubernetes provides a rich authorization plugin interface to support various authorization modules,
such as ABAC, RBAC, Webhook and Node.

## Enable Authorization

Before the release of v0.3.0, kube authorization is disabled by default, you need to enable it manually.

Starting from v0.3.0, Admission is enabled by default.

If you want to disable kube authorization, you need with `--kube-authorization=false` flag when creating cluster.
