#!/usr/bin/env bash
# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DIR="$(dirname "${BASH_SOURCE[0]}")"

DIR="$(realpath "${DIR}")"

ROOT_DIR="$(realpath "${DIR}/../..")"

VERSION="$("${ROOT_DIR}/hack/get-version.sh")"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

SUPPORTED_RELEASES="$(cat "${ROOT_DIR}/supported_releases.txt" | head -n 6)"

LAST_KUBE_RELEASE="$(cat "${ROOT_DIR}/supported_releases.txt" | head -n 1)"

IMAGE_PREFIX=image-prefix
PREFIX=prefix

function want_build() {
  cat <<EOF
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwok ./cmd/kwok
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwokctl ./cmd/kwokctl
EOF
}

function want_build_with_push_bucket() {
  cat <<EOF
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwok ./cmd/kwok
gsutil cp -P ./bin/${GOOS}/${GOARCH}/kwok bucket/releases/${VERSION}/bin/${GOOS}/${GOARCH}/kwok
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/${GOOS}/${GOARCH}/kwokctl bucket/releases/${VERSION}/bin/${GOOS}/${GOARCH}/kwokctl
EOF
}

function want_build_with_push_bucket_staging() {
  cat <<EOF
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwok ./cmd/kwok
gsutil cp -P ./bin/${GOOS}/${GOARCH}/kwok bucket/releases/${PREFIX}-${VERSION}/bin/${GOOS}/${GOARCH}/kwok
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/${GOOS}/${GOARCH}/kwokctl bucket/releases/${PREFIX}-${VERSION}/bin/${GOOS}/${GOARCH}/kwokctl
EOF
}

function want_build_with_push_ghrelease() {
  cat <<EOF
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwok ./cmd/kwok
cp ./bin/${GOOS}/${GOARCH}/kwok kwok-${GOOS}-${GOARCH}
gh -R ghrelease release upload ${VERSION} kwok-${GOOS}-${GOARCH}
GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/${GOOS}/${GOARCH}/kwokctl ./cmd/kwokctl
cp ./bin/${GOOS}/${GOARCH}/kwokctl kwokctl-${GOOS}-${GOARCH}
gh -R ghrelease release upload ${VERSION} kwokctl-${GOOS}-${GOARCH}
EOF
}

function want_image() {
  cat <<EOF
docker buildx build --tag=kwok:${VERSION} --platform=linux/amd64 --load -f ./images/kwok/Dockerfile .
EOF
}

function want_image_nerdctl() {
  cat <<EOF
nerdctl build --tag=kwok:${VERSION} --platform=linux/amd64 -f ./images/kwok/Dockerfile .
EOF
}

function want_image_with_push() {
  cat <<EOF
docker buildx build --tag=kwok:${VERSION} --platform=linux/amd64 --push -f ./images/kwok/Dockerfile .
EOF
}

function want_image_nerdctl_with_push() {
  cat <<EOF
nerdctl build --tag=kwok:${VERSION} --platform=linux/amd64 -f ./images/kwok/Dockerfile .
nerdctl push --platform=linux/amd64 kwok:${VERSION}
EOF
}

function want_image_with_push_staging() {
  cat <<EOF
docker buildx build --tag=${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION} --platform=linux/amd64 --push -f ./images/kwok/Dockerfile .
EOF
}

function want_image_nerdctl_with_push_staging() {
  cat <<EOF
nerdctl build --tag=${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION} --platform=linux/amd64 -f ./images/kwok/Dockerfile .
nerdctl push --platform=linux/amd64 ${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION}
EOF
}

function want_cluster_image() {
  cat <<EOF
docker buildx build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=cluster:${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 --load -f ./images/cluster/Dockerfile .
EOF
}

function want_cluster_image_nerdctl() {
  cat <<EOF
nerdctl build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=cluster:${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 -f ./images/cluster/Dockerfile .
EOF
}

function want_cluster_image_with_push() {
  cat <<EOF
docker buildx build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=cluster:${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 --push -f ./images/cluster/Dockerfile .
EOF
}

function want_cluster_image_nerdctl_with_push() {
  cat <<EOF
nerdctl build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=cluster:${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 -f ./images/cluster/Dockerfile .
nerdctl push --platform=linux/amd64 cluster:${VERSION}-k8s.v${LAST_KUBE_RELEASE}
EOF
}

function want_cluster_image_with_push_staging() {
  cat <<EOF
docker buildx build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 --push -f ./images/cluster/Dockerfile .
EOF
}

function want_cluster_image_nerdctl_with_push_staging() {
  cat <<EOF
nerdctl build --build-arg=kube_version=v${LAST_KUBE_RELEASE} --tag=${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${LAST_KUBE_RELEASE} --platform=linux/amd64 -f ./images/cluster/Dockerfile .
nerdctl push --platform=linux/amd64 ${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${LAST_KUBE_RELEASE}
EOF
}

function want_cross_build() {
  cat <<EOF
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwok ./cmd/kwok
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwokctl ./cmd/kwokctl
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwok ./cmd/kwok
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwokctl ./cmd/kwokctl
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwok ./cmd/kwok
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwokctl ./cmd/kwokctl
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwok ./cmd/kwok
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwokctl ./cmd/kwokctl
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwok.exe ./cmd/kwok
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwokctl.exe ./cmd/kwokctl
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwok.exe ./cmd/kwok
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwokctl.exe ./cmd/kwokctl
EOF
}

function want_cross_build_with_push_bucket() {
  cat <<EOF
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwok ./cmd/kwok
gsutil cp -P ./bin/linux/amd64/kwok bucket/releases/${VERSION}/bin/linux/amd64/kwok
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/linux/amd64/kwokctl bucket/releases/${VERSION}/bin/linux/amd64/kwokctl
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwok ./cmd/kwok
gsutil cp -P ./bin/linux/arm64/kwok bucket/releases/${VERSION}/bin/linux/arm64/kwok
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/linux/arm64/kwokctl bucket/releases/${VERSION}/bin/linux/arm64/kwokctl
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwok ./cmd/kwok
gsutil cp -P ./bin/darwin/amd64/kwok bucket/releases/${VERSION}/bin/darwin/amd64/kwok
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/darwin/amd64/kwokctl bucket/releases/${VERSION}/bin/darwin/amd64/kwokctl
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwok ./cmd/kwok
gsutil cp -P ./bin/darwin/arm64/kwok bucket/releases/${VERSION}/bin/darwin/arm64/kwok
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/darwin/arm64/kwokctl bucket/releases/${VERSION}/bin/darwin/arm64/kwokctl
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwok.exe ./cmd/kwok
gsutil cp -P ./bin/windows/amd64/kwok.exe bucket/releases/${VERSION}/bin/windows/amd64/kwok.exe
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwokctl.exe ./cmd/kwokctl
gsutil cp -P ./bin/windows/amd64/kwokctl.exe bucket/releases/${VERSION}/bin/windows/amd64/kwokctl.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwok.exe ./cmd/kwok
gsutil cp -P ./bin/windows/arm64/kwok.exe bucket/releases/${VERSION}/bin/windows/arm64/kwok.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwokctl.exe ./cmd/kwokctl
gsutil cp -P ./bin/windows/arm64/kwokctl.exe bucket/releases/${VERSION}/bin/windows/arm64/kwokctl.exe
EOF
}

function want_cross_build_with_push_bucket_staging() {
  cat <<EOF
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwok ./cmd/kwok
gsutil cp -P ./bin/linux/amd64/kwok bucket/releases/${PREFIX}-${VERSION}/bin/linux/amd64/kwok
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/linux/amd64/kwokctl bucket/releases/${PREFIX}-${VERSION}/bin/linux/amd64/kwokctl
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwok ./cmd/kwok
gsutil cp -P ./bin/linux/arm64/kwok bucket/releases/${PREFIX}-${VERSION}/bin/linux/arm64/kwok
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/linux/arm64/kwokctl bucket/releases/${PREFIX}-${VERSION}/bin/linux/arm64/kwokctl
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwok ./cmd/kwok
gsutil cp -P ./bin/darwin/amd64/kwok bucket/releases/${PREFIX}-${VERSION}/bin/darwin/amd64/kwok
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/darwin/amd64/kwokctl bucket/releases/${PREFIX}-${VERSION}/bin/darwin/amd64/kwokctl
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwok ./cmd/kwok
gsutil cp -P ./bin/darwin/arm64/kwok bucket/releases/${PREFIX}-${VERSION}/bin/darwin/arm64/kwok
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwokctl ./cmd/kwokctl
gsutil cp -P ./bin/darwin/arm64/kwokctl bucket/releases/${PREFIX}-${VERSION}/bin/darwin/arm64/kwokctl
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwok.exe ./cmd/kwok
gsutil cp -P ./bin/windows/amd64/kwok.exe bucket/releases/${PREFIX}-${VERSION}/bin/windows/amd64/kwok.exe
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwokctl.exe ./cmd/kwokctl
gsutil cp -P ./bin/windows/amd64/kwokctl.exe bucket/releases/${PREFIX}-${VERSION}/bin/windows/amd64/kwokctl.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwok.exe ./cmd/kwok
gsutil cp -P ./bin/windows/arm64/kwok.exe bucket/releases/${PREFIX}-${VERSION}/bin/windows/arm64/kwok.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwokctl.exe ./cmd/kwokctl
gsutil cp -P ./bin/windows/arm64/kwokctl.exe bucket/releases/${PREFIX}-${VERSION}/bin/windows/arm64/kwokctl.exe
EOF
}

function want_cross_build_with_push_ghrelease() {
  cat <<EOF
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwok ./cmd/kwok
cp ./bin/linux/amd64/kwok kwok-linux-amd64
gh -R ghrelease release upload ${VERSION} kwok-linux-amd64
GOOS=linux GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/amd64/kwokctl ./cmd/kwokctl
cp ./bin/linux/amd64/kwokctl kwokctl-linux-amd64
gh -R ghrelease release upload ${VERSION} kwokctl-linux-amd64
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwok ./cmd/kwok
cp ./bin/linux/arm64/kwok kwok-linux-arm64
gh -R ghrelease release upload ${VERSION} kwok-linux-arm64
GOOS=linux GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/linux/arm64/kwokctl ./cmd/kwokctl
cp ./bin/linux/arm64/kwokctl kwokctl-linux-arm64
gh -R ghrelease release upload ${VERSION} kwokctl-linux-arm64
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwok ./cmd/kwok
cp ./bin/darwin/amd64/kwok kwok-darwin-amd64
gh -R ghrelease release upload ${VERSION} kwok-darwin-amd64
GOOS=darwin GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/amd64/kwokctl ./cmd/kwokctl
cp ./bin/darwin/amd64/kwokctl kwokctl-darwin-amd64
gh -R ghrelease release upload ${VERSION} kwokctl-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwok ./cmd/kwok
cp ./bin/darwin/arm64/kwok kwok-darwin-arm64
gh -R ghrelease release upload ${VERSION} kwok-darwin-arm64
GOOS=darwin GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/darwin/arm64/kwokctl ./cmd/kwokctl
cp ./bin/darwin/arm64/kwokctl kwokctl-darwin-arm64
gh -R ghrelease release upload ${VERSION} kwokctl-darwin-arm64
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwok.exe ./cmd/kwok
cp ./bin/windows/amd64/kwok.exe kwok-windows-amd64.exe
gh -R ghrelease release upload ${VERSION} kwok-windows-amd64.exe
GOOS=windows GOARCH=amd64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/amd64/kwokctl.exe ./cmd/kwokctl
cp ./bin/windows/amd64/kwokctl.exe kwokctl-windows-amd64.exe
gh -R ghrelease release upload ${VERSION} kwokctl-windows-amd64.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwok.exe ./cmd/kwok
cp ./bin/windows/arm64/kwok.exe kwok-windows-arm64.exe
gh -R ghrelease release upload ${VERSION} kwok-windows-arm64.exe
GOOS=windows GOARCH=arm64 go build -ldflags '-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION} -X sigs.k8s.io/kwok/pkg/consts.KubeVersion=v${LAST_KUBE_RELEASE}' -o ./bin/windows/arm64/kwokctl.exe ./cmd/kwokctl
cp ./bin/windows/arm64/kwokctl.exe kwokctl-windows-arm64.exe
gh -R ghrelease release upload ${VERSION} kwokctl-windows-arm64.exe
EOF
}

function want_cross_image() {
  cat <<EOF
docker buildx build --tag=kwok:${VERSION} --platform=linux/amd64 --platform=linux/arm64 --load -f ./images/kwok/Dockerfile .
EOF
}

function want_cross_image_nerdctl() {
  cat <<EOF
nerdctl build --tag=kwok:${VERSION} --platform=linux/amd64 --platform=linux/arm64 -f ./images/kwok/Dockerfile .
EOF
}

function want_cross_image_with_push() {
  cat <<EOF
docker buildx build --tag=kwok:${VERSION} --platform=linux/amd64 --platform=linux/arm64 --push -f ./images/kwok/Dockerfile .
EOF
}

function want_cross_image_nerdctl_with_push() {
  cat <<EOF
nerdctl build --tag=kwok:${VERSION} --platform=linux/amd64 --platform=linux/arm64 -f ./images/kwok/Dockerfile .
nerdctl push --platform=linux/amd64 --platform=linux/arm64 kwok:${VERSION}
EOF
}

function want_cross_image_with_push_staging() {
  cat <<EOF
docker buildx build --tag=${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION} --platform=linux/amd64 --platform=linux/arm64 --push -f ./images/kwok/Dockerfile .
EOF
}

function want_cross_image_nerdctl_with_push_staging() {
  cat <<EOF
nerdctl build --tag=${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION} --platform=linux/amd64 --platform=linux/arm64 -f ./images/kwok/Dockerfile .
nerdctl push --platform=linux/amd64 --platform=linux/arm64 ${IMAGE_PREFIX}/kwok:${PREFIX}-${VERSION}
EOF
}

function want_cross_cluster_image() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "docker buildx build --build-arg=kube_version=v${v} --tag=cluster:${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 --load -f ./images/cluster/Dockerfile ."
  done
}

function want_cross_cluster_image_nerdctl() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "nerdctl build --build-arg=kube_version=v${v} --tag=cluster:${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 -f ./images/cluster/Dockerfile ."
  done
}

function want_cross_cluster_image_with_push() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "docker buildx build --build-arg=kube_version=v${v} --tag=cluster:${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 --push -f ./images/cluster/Dockerfile ."
  done
}

function want_cross_cluster_image_nerdctl_with_push() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "nerdctl build --build-arg=kube_version=v${v} --tag=cluster:${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 -f ./images/cluster/Dockerfile ."
    echo "nerdctl push --platform=linux/amd64 --platform=linux/arm64 cluster:${VERSION}-k8s.v${v}"
  done
}

function want_cross_cluster_image_with_push_staging() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "docker buildx build --build-arg=kube_version=v${v} --tag=${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 --push -f ./images/cluster/Dockerfile ."
  done
}

function want_cross_cluster_image_nerdctl_with_push_staging() {
  for v in ${SUPPORTED_RELEASES} ; do
    echo "nerdctl build --build-arg=kube_version=v${v} --tag=${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${v} --platform=linux/amd64 --platform=linux/arm64 -f ./images/cluster/Dockerfile ."
    echo "nerdctl push --platform=linux/amd64 --platform=linux/arm64 ${IMAGE_PREFIX}/cluster:${PREFIX}-${VERSION}-k8s.v${v}"
  done
}

function main() {
  failed=()
  export DRY_RUN=true
  make --no-print-directory -C "${ROOT_DIR}" build | diff -u <(want_build) - || failed+=("build")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket build | diff -u <(want_build_with_push_bucket) - || failed+=("build-with-push-bucket")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} build | diff -u <(want_build_with_push_bucket_staging) - || failed+=("build-with-push-bucket-staging")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true GH_RELEASE=ghrelease build | diff -u <(want_build_with_push_ghrelease) - || failed+=("build-with-push-ghrelease")

  make --no-print-directory -C "${ROOT_DIR}" image | diff -u <(want_image) - || failed+=("image")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true image | diff -u <(want_image_with_push) - || failed+=("image-with-push")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} image | diff -u <(want_image_with_push_staging) - || failed+=("image-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl image | diff -u <(want_image_nerdctl) - || failed+=("image-nerdctl")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true image | diff -u <(want_image_nerdctl_with_push) - || failed+=("image-nerdctl-with-push")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} image | diff -u <(want_image_nerdctl_with_push_staging) - || failed+=("image--nerdctl-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" cluster-image | diff -u <(want_cluster_image) - || failed+=("cluster-image")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true cluster-image | diff -u <(want_cluster_image_with_push) - || failed+=("cluster-image-with-push")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cluster-image | diff -u <(want_cluster_image_with_push_staging) - || failed+=("cluster-image-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl cluster-image | diff -u <(want_cluster_image_nerdctl) - || failed+=("cluster-image-nerdctl")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true cluster-image | diff -u <(want_cluster_image_nerdctl_with_push) - || failed+=("cluster-image-nerdctl-with-push")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cluster-image | diff -u <(want_cluster_image_nerdctl_with_push_staging) - || failed+=("cluster-image-nerdctl-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" cross-build | diff -u <(want_cross_build) - || failed+=("cross-build")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket cross-build | diff -u <(want_cross_build_with_push_bucket) - || failed+=("cross-build-with-push-bucket")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} cross-build | diff -u <(want_cross_build_with_push_bucket_staging) - || failed+=("cross-build-with-push-bucket-staging")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true GH_RELEASE=ghrelease cross-build | diff -u <(want_cross_build_with_push_ghrelease) - || failed+=("cross-build-with-push-ghrelease")

  make --no-print-directory -C "${ROOT_DIR}" cross-image | diff -u <(want_cross_image) - || failed+=("cross-image")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true cross-image | diff -u <(want_cross_image_with_push) - || failed+=("cross-image-with-push")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cross-image | diff -u <(want_cross_image_with_push_staging) - || failed+=("cross-image-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl cross-image | diff -u <(want_cross_image_nerdctl) - || failed+=("cross-image-nerdctl")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true cross-image | diff -u <(want_cross_image_nerdctl_with_push) - || failed+=("cross-image-nerdctl-with-push")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cross-image | diff -u <(want_cross_image_nerdctl_with_push_staging) - || failed+=("cross-image-nerdctl-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" cross-cluster-image | diff -u <(want_cross_cluster_image) - || failed+=("cross-cluster-image")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true cross-cluster-image | diff -u <(want_cross_cluster_image_with_push) - || failed+=("cross-cluster-image-with-push")
  make --no-print-directory -C "${ROOT_DIR}" PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cross-cluster-image | diff -u <(want_cross_cluster_image_with_push_staging) - || failed+=("cross-cluster-image-with-push-staging")

  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl cross-cluster-image | diff -u <(want_cross_cluster_image_nerdctl) - || failed+=("cross-cluster-image-nerdctl")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true cross-cluster-image | diff -u <(want_cross_cluster_image_nerdctl_with_push) - || failed+=("cross-cluster-image-nerdctl-with-push")
  make --no-print-directory -C "${ROOT_DIR}" BUILDER=nerdctl PUSH=true BUCKET=bucket STAGING=true STAGING_PREFIX=${PREFIX} STAGING_IMAGE_PREFIX=${IMAGE_PREFIX} cross-cluster-image | diff -u <(want_cross_cluster_image_nerdctl_with_push_staging) - || failed+=("cross-cluster-image-nerdctl-with-push-staging")

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

main
