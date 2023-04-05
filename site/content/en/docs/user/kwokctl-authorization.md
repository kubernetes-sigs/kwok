# `kwokctl` Authorization

{{< hint "info" >}}

This document walks you known about kubernetes authorization in kwokctl.

{{< /hint >}}

## What is Authorization

Kubernetes provides a rich authorization plugin interface to support various authorization modules,
such as ABAC, RBAC, Webhook and Node.

In kwokctl, we don't care about it, so that default will disable kube authorization.

## Enable Authorization

On Kind runtime is enabled by default, but on other runtime, you need to enable it manually.

If you want to enable kube authorization, you need with `--kube-authorization` flag when create cluster.
