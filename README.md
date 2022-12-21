# `KWOK` (`K`ubernetes-`W`ith`O`ut-`K`ubelet)

<img align="right" width="180px" src="./logo/kwok.svg"/>

The repository is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employes
a pretty low resource footprint that you can easily play around on your laptop.

So far we provide two tools:

- **Kwok:** Core of this repo. It simulates thousands of fake Nodes.
- **Kwokctl:** A CLI to facilitate creating and managing clusters simulated by Kwok.

## Get started

The following examples are tested to work with the latest version of Kwok/Kwokctl.

### Basic Usage

- [Kwok Manage Nodes and Pods](./docs/examples/kwok-manage-nodes-and-pods.md) - Kwok's basic management of Node and Pod operations
- Kwok - maintain Nodes heartbeat and Pods status.
  - [Kwok in Cluster](./docs/examples/kwok-in-cluster.md) - Installing Kwok in a cluster
  - [Kwok in Local](./docs/examples/kwok-in-local.md) - Run Kwok in the local for a cluster
- Kwokctl - cluster creation, etcd snapshot, etc.
  - [Kwokctl Manage Clusters](./docs/examples/kwokctl-manage-cluster.md) - Create/Delete a cluster in local where all nodes are managed by Kwok
  - [Kwokctl Snapshot Cluster](./docs/examples/kwokctl-snapshot.md) - Save/Restore the Etcd data of a cluster created by Kwokctl

``` console
$ time kwokctl create cluster
Creating cluster "kwok-kwok"
Starting cluster "kwok-kwok"
Cluster "kwok-kwok" is ready
You can now use your cluster with:

    kubectl config use-context kwok-kwok

Thanks for using kwok!

real    0m2.599s
user    0m0.606s
sys     0m0.254s
```

### Actual Usage

If you are using Kwok/Kubectl as a testing or CI in your project and would like to share your experience with others, then please add your example below

<!--
Add your examples like

- [Example name](./docs/examples/example-name/example-name.md) - Example description
-->

- [TBD](#) - Example description

## Kwokctl Runtime and OS Support

Runtime indicates which medium kwokctl will use to start the cluster

|         | binary | docker | kind | nerdctl | podman |
| ------- | ------ | ------ | ---- | ------- | ------ |
| Linux   | 🟢      | 🟢      | 🟢    | 🔵       | ⚫      |
| Darwin  | 🟠      | 🟢      | 🟢    | 🔴       | 🔴      |
| Windows | 🟠/🔵    | 🔵      | 🔵    | 🔴       | 🔴      |

- 🟢 Supported
- 🔴 Not supported
- 🟠 Need to use your own build of the Kubernetes binary
- 🔵 Expected support but not fully tested
- ⚫ TODO

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
