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

KIND_VERSION=0.14.0

KUBE_VERSION=1.24.2

BUILDX_VERSION=0.8.2

COMPOSE_VERSION=2.10.1

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
  ln -s /usr/local/lib/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud
  ln -s /usr/local/lib/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil
  gcloud info
  gcloud config list
  gcloud auth list
}

function install_gsutil() {
  if command_exist gsutil; then
    return 0
  fi

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

  curl -SL -o "/usr/local/bin/kind" "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-$(runtime_os)-$(runtime_arch)" &&
    chmod +x "/usr/local/bin/kind"

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
  curl -SL -o "/usr/local/bin/kubectl" "https://dl.k8s.io/release/v${KUBE_VERSION}/bin/$(runtime_os)/$(runtime_arch)/kubectl" &&
    chmod +x "/usr/local/bin/kubectl"

  if ! command_exist kubectl; then
    echo kind is installed but not effective >&2
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

function install_compose() {
  local binary

  if docker compose version; then
    return 0
  fi

  binary="$(runtime_home)/.docker/cli-plugins/docker-compose"

  mkdir -p "$(dirname "${binary}")" &&
    wget -O "${binary}" "https://github.com/docker/compose/releases/download/v${COMPOSE_VERSION}/docker-compose-$(runtime_os)-$(runtime_arch_alias)" &&
    chmod +x "${binary}"

  if ! docker compose version; then
    echo docker-compose is installed but not effective >&2
    return 1
  fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  function usage() {
    local requirements=(
      gsutil
      kubectl
      kind
      buildx
      compose
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
