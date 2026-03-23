---
title: "Scale with Custom Resource"
---

# Scale with Custom Resource

{{< hint "info" >}}

This document walks you through how to scale custom Kubernetes resources using `kwokctl scale`.

{{< /hint >}}

The `kwokctl scale` command supports scaling custom resource types beyond the built-in `node` and `pod`
types. Any resource can be scaled by defining a `KwokctlResource` configuration that provides a name,
default parameters, and a Go template for the resource manifest.

## Create a Cluster

``` bash
kwokctl create cluster
```

## Create Nodes

``` bash
kwokctl scale node --replicas 2
```

## Define a Deployment Resource

Create a `KwokctlResource` manifest that acts as the template for Deployments.
Save the following content as `deployment-resource.yaml`:

{{< expand "deployment-resource.yaml" >}}

{{< code-sample file="deployment-resource.yaml" >}}

{{< /expand >}}

The `template` field uses Go template syntax. The built-in functions `Name`, `Namespace`, and `Index`
are available, where `Name` is the generated resource name, `Namespace` is the value of the
`--namespace` flag (falling back to `"default"`), and `Index` is the zero-based replica index.

## Scale Deployments

Use `--config` to load the resource definition and `--replicas` to set the number of Deployments to create:

``` bash
kwokctl scale deployment --replicas 5 --config ./deployment-resource.yaml
```

Use `--param` to override default template parameters at runtime. For example, set each Deployment to
have 3 replicas:

``` bash
kwokctl scale deployment --replicas 5 --param '.replicas=3' --config ./deployment-resource.yaml
```

### Persist the Resource Definition

Instead of passing `--config` every time, you can append the `KwokctlResource` to the default
configuration file (`~/.kwok/kwok.yaml`) so that `kwokctl scale deployment` works without any
extra flags:

``` bash
cat deployment-resource.yaml >> ~/.kwok/kwok.yaml
kwokctl scale deployment --replicas 5 --param '.replicas=3'
```

## Scale Deployments into a Specific Namespace

Create the target namespace first, then pass it with `--namespace`:

``` bash
kubectl create namespace my-namespace
kwokctl scale deployment --replicas 5 --namespace my-namespace --param '.replicas=3' --config ./deployment-resource.yaml
```

## Verify

``` bash
kubectl get deployments -A
```

``` bash
kubectl get pods -A
```

The output should show `deployment-000000` through `deployment-000004` (five deployments), each
with the pods spawned by the Kubernetes Deployment controller:

``` log
NAMESPACE   NAME                READY   UP-TO-DATE   AVAILABLE   AGE
default     deployment-000000   3/3     3            3           5s
default     deployment-000001   3/3     3            3           5s
default     deployment-000002   3/3     3            3           5s
default     deployment-000003   3/3     3            3           5s
default     deployment-000004   3/3     3            3           5s
```

## Delete the Cluster

``` bash
kwokctl delete cluster
```

[kwokctl]: https://kwok.sigs.k8s.io/docs/user/install
[kubectl]: https://kubernetes.io/docs/tasks/tools/
