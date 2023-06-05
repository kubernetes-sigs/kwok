# Platform-Specific Kubernetes Binaries

{{< hint "info" >}}

This document provides details on how to build Kubernetes binaries and run `kwokctl` locally.

{{< /hint >}}

## Overview

When running `kwokctl` with `binary` runtime (`kwokctl create cluster --runtime=binary`),
`kwokctl` will download Kubernetes binaries from [dl.k8s.io]/[www.downloadkubernetes.com] and use them to create cluster.
but that only works on Linux.

## For Non-Linux

Building Kubernetes Binaries and setting up `kwokctl` defaults to use them

``` bash
KUBE_VERSION="v1.27.1"
SRC_DIR="${HOME}/.kwok/cache/kubernetes/${KUBE_VERSION}"
mkdir -p "${SRC_DIR}" && cd "${SRC_DIR}" &&
wget "https://dl.k8s.io/${KUBE_VERSION}/kubernetes-src.tar.gz" -O - | tar xz &&
make WHAT="cmd/kube-apiserver cmd/kube-controller-manager cmd/kube-scheduler" &&
cat <<EOF >> ~/.kwok/kwok.yaml
---
kind: KwokctlConfiguration
apiVersion: config.kwok.x-k8s.io/v1alpha1
options:
  kubeBinaryPrefix: $(pwd)/_output/local/bin/$(go env GOOS)/$(go env GOARCH)
---
EOF
```

The binaries will be located in `~/.kwok/cache/kubernetes/${KUBE_VERSION}/_output/local/bin/$(go env GOOS)/$(go env GOARCH)`.

Now, we can create cluster using `kwokctl` with `binary` runtime.

``` bash
kwokctl create cluster \
  --runtime=binary
```

[dl.k8s.io]: https://dl.k8s.io
[www.downloadkubernetes.com]: https://www.downloadkubernetes.com
