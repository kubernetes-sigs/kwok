---
title: "Auditing"
---

# `kwokctl` Auditing

{{< hint "info" >}}

This document walks you through how to enable [Audit policy](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#audit-policy) on a `kwokctl` cluster

{{< /hint >}}

## Setup Audit Policy

``` bash
cat <<EOF > audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
EOF
```

## Create a cluster with audit policy

``` bash
kwokctl create cluster --kube-audit-policy audit-policy.yaml
```

## Get audit logs

``` bash
kwokctl logs audit
```

## Example audit logs

<img width="700px" src="/img/demo/audit-log.svg">

