# Getting Started

{{< hint "info" >}}

This document provides details on how to contribute to the project.

{{< /hint >}}

## Kubernetes Community Guidelines

- [Kubernetes Contributor Guide]
- [Contributor Cheat Sheet]

## Requirements

Our source code is written in [golang] and managed with [git].

On create clusters and builds you will need to [install docker].

## Reaching out

Please check [the issues] or [the discussions] to see if there is any existing discussion or work related to your interests.

In particular, if you're just getting started, you may want to look for issues labeled
{{< ghlabel background="#7057ff" href="https://github.com/kubernetes-sigs/kwok/labels/good first issue" >}}good first issue{{< /ghlabel >}}
or
{{< ghlabel background="#006b75" href="https://github.com/kubernetes-sigs/kwok/labels/help wanted" >}}help wanted{{< /ghlabel >}}
which are standard labels in the Kubernetes project.

If you're interested in working on any of these, leave a comment to let us know!

If you do not see anything, please file a [new issue] or [new discussion].

{{< hint "warning" >}}

**NOTE** Please file an issue or discussion before starting work on a new feature or large change.

{{< /hint >}}

The maintainers of this project are reachable via:

- [Kubernetes Slack] in [#kwok], [#kwok-dev] or [#sig-scheduling] channel
- The issue tracker by [filing an issue][new issue]
- The discussion tracker by [filing a discussion][new discussion]
- The Kubernetes [SIG-Scheduling] [Mailing List]

Current maintainers in [owner list], feel free to reach out directly if you have any questions!

## Next Steps

If you're planning to contribute code changes, you'll want to read the [development] next.

If you're looking to contribute documentation improvements, you'll want to read the [documentation] next.

[git]: https://git-scm.com/downloads
[the issues]: https://github.com/kubernetes-sigs/kwok/issues
[the discussions]: https://github.com/kubernetes-sigs/kwok/discussions
[new issue]: https://github.com/kubernetes-sigs/kwok/issues/new/choose
[new discussion]: https://github.com/kubernetes-sigs/kwok/discussions/new/choose
[golang]: https://golang.org/doc/install
[install docker]: https://docs.docker.com/install/#supported-platforms
[Kubernetes Slack]: https://slack.k8s.io/
[#sig-scheduling]: https://kubernetes.slack.com/messages/sig-scheduling/
[#kwok]: https://kubernetes.slack.com/messages/kwok/
[#kwok-dev]: https://kubernetes.slack.com/messages/kwok-dev/
[SIG-Scheduling]: https://github.com/kubernetes/community/blob/master/sig-scheduling/README.md
[Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling
[Kubernetes Contributor Guide]: https://git.k8s.io/community/contributors/guide
[Contributor Cheat Sheet]: https://git.k8s.io/community/contributors/guide/contributor-cheatsheet
[owner list]: https://github.com/kubernetes-sigs/kwok/blob/main/OWNERS
[development]: {{< relref "/docs/contributing/development" >}}
[documentation]: {{< relref "/docs/contributing/documentation" >}}
