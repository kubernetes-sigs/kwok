# Install

{{< hint "info" >}}

This document walks through the installation of `kwokctl` and `kwok` binaries.

{{< /hint >}}

## Homebrew

On Linux/MacOS systems you can install kwok/kwokctl via [brew](https://formulae.brew.sh/formula/kwok):

this will install the latest version of `kwokctl` and `kwok` binaries.

``` bash
brew install kwok
```

## Go Install

also, you can install `kwokctl` and `kwok` binaries via [golang].

``` bash
go install sigs.k8s.io/kwok/cmd/{kwok,kwokctl}@latest
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
