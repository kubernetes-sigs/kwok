# Run Kwok in the Local

This doc walks you through how to run `kwok` in the local.

## Variables preparation

``` bash
# Kwok repository to download image from
KWOK_REPO=kubernetes-sigs/kwok
# Get latest Kwok binary
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Install Kwok

Firstly, we download and install Kwok 

``` bash
wget -O kwok -c "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok-$(go env GOOS)-$(go env GOARCH)"
chmod +x kwok
sudo mv kwok /usr/local/bin/kwok
```

## Run Kwok in the local

Finally, we're able to run `kwok` in the local for a cluster and maintain their heartbeats:

``` bash
kwok \
  --kubeconfig=~/.kube/config \
  --manage-all-nodes=false \
  --manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake \
  --manage-nodes-with-label-selector= \
  --disregard-status-with-annotation-selector=kwok.x-k8s.io/status=custom \
  --disregard-status-with-label-selector= \
  --cidr=10.0.0.1/24 \
  --node-ip=10.0.0.1
```
