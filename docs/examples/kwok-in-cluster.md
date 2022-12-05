# Deploy Kwok in a Cluster

This document walks you through how to deploy `kwok` in a Kubernetes cluster.

## Variables preparation

``` bash
# Temporary directory
KWOK_WORK_DIR=$(mktemp -d)
# Kwok repository to download image from
KWOK_REPO=kubernetes-sigs/kwok
# Get latest Kwok image
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```

## Render kustomization yaml

Firstly, generate a kustomization template yaml to the previously-defined temporary directory.

``` bash
cat <<EOF > "${KWOK_WORK_DIR}/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: registry.k8s.io/kwok/kwok
    newTag: "${KWOK_LATEST_RELEASE}"
resources:
  - "https://github.com/${KWOK_REPO}/kustomize/kwok?ref=${KWOK_LATEST_RELEASE}"
EOF
```

Next, render it with the prepared variables.

``` bash
kubectl kustomize "${KWOK_WORK_DIR}" > "${KWOK_WORK_DIR}/kwok.yaml"
```

## Kwok deployment

Finally, we're able to deploy `kwok`:

``` bash
kubectl apply -f "${KWOK_WORK_DIR}/kwok.yaml"
```
