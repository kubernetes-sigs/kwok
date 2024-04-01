---
title: Installation
aliases:
  - /docs/user/install/
---

# Installation

{{< hint "info" >}}

This document walks through the installation of `kwokctl` and `kwok` binaries.

{{< /hint >}}

## Install with Package Manager

This will install the latest version of `kwokctl` and `kwok` binaries.

{{< tabs "install-with-package-manager" >}}

{{< tab "Homebrew" >}}

On Linux/MacOS systems you can install kwok/kwokctl via [brew](https://formulae.brew.sh/formula/kwok):

``` bash
brew install kwok
```

{{< /tab >}}

{{< tab "WinGet" >}}

On Windows systems you can install kwok/kwokctl via winget:

``` bash
winget install --id=Kubernetes.kwok -e
winget install --id=Kubernetes.kwokctl -e
```

{{< /tab >}}

{{< /tabs >}}

## Install with Golang

Also, you can install `kwokctl` and `kwok` binaries via [golang].

``` bash
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')

go install sigs.k8s.io/kwok/cmd/{kwok,kwokctl}@${KWOK_LATEST_RELEASE}
```

## Binary Releases

Or download from github releases page:

### Variables preparation

``` bash
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

### Install `kwokctl`

``` bash
wget -O kwokctl -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwokctl-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwokctl
sudo mv kwokctl /usr/local/bin/kwokctl
```

### Install `kwok`

``` bash
wget -O kwok -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwok
sudo mv kwok /usr/local/bin/kwok
```

[golang]: https://golang.org/doc/install
