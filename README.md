# `KWOK` (`K`ubernetes `W`ith`O`ut `K`ubelet)

<img align="right" width="180px" src="./logo/kwok.svg"/>

[KWOK] is a toolkit that enables setting up a cluster of thousands of Nodes in seconds.
Under the scene, all Nodes are simulated to behave like real ones, so the overall approach employs
a pretty low resource footprint that you can easily play around on your laptop.

So far we provide two tools:

- **kwok:** Core of this repo. It simulates thousands of fake Nodes.
- **kwokctl:** A CLI to facilitate creating and managing clusters simulated by Kwok.

Please see [our website] for more in-depth information.

<img width="700px" src="./demo/manage-clusters.svg">

## Community

See our own [contributor guide] and the Kubernetes [community page].

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct][code of conduct].

[KWOK]: https://sigs.k8s.io/kwok
[our website]: https://kwok.sigs.k8s.io
[community page]: https://kubernetes.io/community/
[contributor guide]: https://kwok.sigs.k8s.io/docs/contributing/getting-started
[code of conduct]: https://github.com/kubernetes-sigs/kwok/blob/main/code-of-conduct.md
