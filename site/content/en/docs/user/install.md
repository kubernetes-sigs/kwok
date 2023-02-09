# Install

{{< hint "info" >}}

This document walks through the installation of `kwokctl` and `kwok` binaries.

{{< /hint >}}

## Homebrew

On Linux/MacOS systems you can install kwok/kwokctl via [brew](https://formulae.brew.sh/formula/kwok):

``` bash
brew install kwok
```

## Binary Releases

### Variables preparation

``` bash
# Kwok repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

### Install Kwokctl

``` bash
wget -O kwokctl -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwokctl-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwokctl
sudo mv kwokctl /usr/local/bin/kwokctl
```

### Install Kwok

``` bash
wget -O kwok -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwok
sudo mv kwok /usr/local/bin/kwok
```
