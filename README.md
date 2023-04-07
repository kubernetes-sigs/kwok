# `KWOK` (`K`ubernetes `W`ith`O`ut `K`ubelet)

<img align="right" width="180px" src="./logo/kwok.svg"/>

[KWOK] is pronounced as `/kwɔk/`.

[KWOK] is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employs
a pretty low resource footprint that you can easily play around on your laptop.

## What is KWOK?

KWOK stands for Kubernetes WithOut Kubelet. So far, it provides two tools:

- `kwok` is the cornerstone of this project, responsible for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.
- `kwokctl` is a CLI tool designed to streamline the creation and management of clusters, with nodes simulated by `kwok`.

Please see [our website] for more in-depth information.

<img width="700px" src="./demo/manage-clusters.svg">

## Why KWOK?

- Lightweight: You can [simulate thousands of nodes](https://kwok.sigs.k8s.io/docs/user/kwok-manage-nodes-and-pods/) on your laptop without significant consumption of CPU or memory resources.
Currently, KWOK can reliably maintain 1k nodes and 100k pods.
- Fast: You can create and delete clusters and nodes almost instantly, without waiting for boot or provisioning.
Currently, KWOK can create 20 nodes or pods per second.
- Compatibility: KWOK works with any tools or clients that are compliant with Kubernetes APIs, such as kubectl, helm, kui, etc.
- Portability: KWOK has no specific hardware or software requirements. You can [run it using pre-built images](https://kwok.sigs.k8s.io/docs/user/all-in-one-image/), once Docker or Nerdctl is installed. Alternatively, binaries are also available for all platforms and can be easily installed.
- Flexibility: You can configure different node types, labels, taints, capacities, conditions, etc., and you can configure different pod behaviors, status, etc. to test different scenarios and edge cases.

## Community

See our own [contributor guide] and the Kubernetes [community page].

### Getting Involved

If you're interested in participating in future discussions or development related to KWOK, there are several ways to get involved:

- Slack: [#kwok] for general usage discussion, [#kwok-dev] for development discussion. (visit [slack.k8s.io] for a workspace invitation)
- Open Issues/PRs/Discussions in [sigs.k8s.io/kwok]

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct][code of conduct].

[KWOK]: https://sigs.k8s.io/kwok
[our website]: https://kwok.sigs.k8s.io
[community page]: https://kubernetes.io/community/
[contributor guide]: https://kwok.sigs.k8s.io/docs/contributing/getting-started
[code of conduct]: https://github.com/kubernetes-sigs/kwok/blob/main/code-of-conduct.md
[sigs.k8s.io/kwok]: https://sigs.k8s.io/kwok/
[#kwok]: https://kubernetes.slack.com/messages/kwok/
[#kwok-dev]: https://kubernetes.slack.com/messages/kwok-dev/
[slack.k8s.io]: https://slack.k8s.io/
