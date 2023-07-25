---
title: "Metrics Server Binaries"
---

# Metrics Server Binary

{{< hint "info" >}}

This document provides details on how to build Kubernetes binaries and run `kwokctl` locally.

{{< /hint >}}

## Build

Building Kubernetes Binaries and setting up `kwokctl` defaults to use them

``` bash
METRICS_SERVER_VERSION="0.6.3"
SRC_DIR="${HOME}/.kwok/cache/metrics-server/v${METRICS_SERVER_VERSION}"
mkdir -p "${SRC_DIR}" && cd "${SRC_DIR}" &&
wget "https://github.com/kubernetes-sigs/metrics-server/archive/refs/tags/v${METRICS_SERVER_VERSION}.tar.gz" -O - | tar xz &&
cd metrics-server-${METRICS_SERVER_VERSION} && make GIT_TAG="v${METRICS_SERVER_VERSION}" metrics-server &&
cat <<EOF >> ~/.kwok/kwok.yaml
---
kind: KwokctlConfiguration
apiVersion: config.kwok.x-k8s.io/v1alpha1
options:
  metricsServerBinary: $(pwd)/metrics-server
---
EOF
```

The binaries will be located in `~/.kwok/cache/metrics-server/${METRICS_SERVER_VERSION}/metrics-server`.
