#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
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

ROOT_DIR="$(realpath "${DIR}/..")"

LOCAL_BIN_DIR="${ROOT_DIR}/bin"

export PATH="${LOCAL_BIN_DIR}:${PATH}"

KIND_VERSION=0.19.0

KUBE_VERSION=1.27.3

# TODO: Stay at 0.9 in figuring out the Attestations of buildx.
# https://github.com/docker/buildx/pull/1412
BUILDX_VERSION=0.9.1

function command_exist() {
  local command="${1}"
  type "${command}" >/dev/null 2>&1
}

function runtime_home() {
  if [[ "${HOME}" != "" ]]; then
    echo "${HOME}"
  elif [[ "${USERPROFILE}" != "" ]]; then
    echo "${USERPROFILE}"
  else
    echo "Failed get home dir" >&2
    exit 1
  fi

}

function runtime_arch() {
  go env GOARCH
}

function runtime_os() {
  go env GOOS
}

function runtime_arch_alias() {
  local arch
  arch="$(runtime_arch)"
  case "${arch}" in
  amd64)
    echo "x86_64"
    ;;
  arm64)
    echo "aarch64"
    ;;
  *)
    echo "${arch}"
    ;;
  esac
}

function _install_gcloud() {
  mkdir -p /usr/local/lib/
  curl https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -o /tmp/google-cloud-sdk.tar.gz
  tar xzf /tmp/google-cloud-sdk.tar.gz -C /usr/local/lib/
  rm /tmp/google-cloud-sdk.tar.gz
  /usr/local/lib/google-cloud-sdk/install.sh \
    --bash-completion=false \
    --usage-reporting=false \
    --quiet
  ln -s /usr/local/lib/google-cloud-sdk/bin/gcloud "${LOCAL_BIN_DIR}/gcloud"
  ln -s /usr/local/lib/google-cloud-sdk/bin/gsutil "${LOCAL_BIN_DIR}/gsutil"
  gcloud info
  gcloud config list
  gcloud auth list
}

function install_gsutil() {
  if command_exist gsutil; then
    return 0
  fi

  mkdir -p "${LOCAL_BIN_DIR}"
  _install_gcloud

  if ! command_exist gsutil; then
    echo gsutil is installed but not effective >&2
    return 1
  fi

  gsutil version
}

function install_kind() {
  if command_exist kind; then
    return 0
  fi

  mkdir -p "${LOCAL_BIN_DIR}"
  curl -SL -o "${LOCAL_BIN_DIR}/kind" "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-$(runtime_os)-$(runtime_arch)" &&
    chmod +x "${LOCAL_BIN_DIR}/kind"

  if ! command_exist kind; then
    echo kind is installed but not effective >&2
    return 1
  fi

  kind version
}

function install_kubectl() {
  if command_exist kubectl; then
    return 0
  fi

  mkdir -p "${LOCAL_BIN_DIR}"
  curl -SL -o "${LOCAL_BIN_DIR}/kubectl" "https://dl.k8s.io/release/v${KUBE_VERSION}/bin/$(runtime_os)/$(runtime_arch)/kubectl" &&
    chmod +x "${LOCAL_BIN_DIR}/kubectl"

  if ! command_exist kubectl; then
    echo kubectl is installed but not effective >&2
    return 1
  fi

  kubectl version --client=true
}

function install_buildx() {
  local binary

  if docker buildx version; then
    return 0
  fi

  binary="$(runtime_home)/.docker/cli-plugins/docker-buildx"

  mkdir -p "$(dirname "${binary}")" &&
    wget -O "${binary}" "https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.$(runtime_os)-$(runtime_arch)" &&
    chmod +x "${binary}"

  if ! docker buildx version; then
    echo docker-buildx is installed but not effective >&2
    return 1
  fi

  if ! docker buildx inspect --builder kwok >/dev/null 2>&1; then
    docker buildx create --use --name kwok >/dev/null 2>&1
  fi
}

function install_kustomize() {
  if command_exist kustomize; then
    return 0
  fi

  mkdir -p "${LOCAL_BIN_DIR}"
  GOBIN="${LOCAL_BIN_DIR}" go install sigs.k8s.io/kustomize/kustomize/v4
  if ! command_exist kustomize; then
    echo kustomize is installed but not effective >&2
    return 1
  fi

  kustomize version
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  function usage() {
    local requirements=(
      gsutil
      kubectl
      kind
      buildx
    )
    echo "Usage: ${0} [flags] [requirements]"
    echo "  Empty argument will install all requirements."
    echo "  REQUIREMENT:"
    for c in "${requirements[@]}"; do
      echo " - ${c}"
    done
  }

  function main() {
    local arg
    while [[ $# -gt 0 ]]; do
      arg="$1"
      case "${arg}" in
      --help)
        usage
        exit 0
        ;;
      -*)
        echo "Unknown argument: ${arg}"
        usage
        exit 1
        ;;
      *)
        "install_${arg}" || exit 1
        shift
        ;;
      esac
    done
  }
  main "${@}"
fi
