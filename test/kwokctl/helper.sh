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

ROOT_DIR="$(realpath "${DIR}/../..")"

source "${ROOT_DIR}/hack/requirements.sh"
source "${DIR}/suite.sh"

VERSION="test"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

LOCAL_PATH="${ROOT_DIR}/bin/${GOOS}/${GOARCH}"

export KWOK_CONTROLLER_BINARY="${LOCAL_PATH}/kwok"
export KWOKCTL_CONTROLLER_BINARY="${LOCAL_PATH}/kwokctl"
export KWOK_CONTROLLER_IMAGE="localhost/kwok:${VERSION}"
export PATH="${LOCAL_PATH}:${PATH}"

function test_all() {
  local runtime="${1}"
  local cases="${2}"
  local releases=("${@:3}")

  echo "Test ${cases} on ${runtime} for ${releases[*]}"
  if KWOK_RUNTIME="${runtime}" "${DIR}/kwokctl_${cases}_test.sh" "${releases[@]}"; then
    rm -rf "${KWOK_LOGS_DIR}"
  else
    return 1
  fi

  for name in $(kwokctl get clusters); do
    echo "Clean up cluster '${name}' that have not been deleted."
    delete_cluster "${name}"
    return 1
  done
}

# Test only the latest releases of Kubernetes
LAST_RELEASE_SIZE="${LAST_RELEASE_SIZE:-1}"

function supported_releases() {
  head <"${ROOT_DIR}/supported_releases.txt" -n "${LAST_RELEASE_SIZE}"
}

function build_kwokctl() {
  if [[ -f "${KWOKCTL_CONTROLLER_BINARY}" ]]; then
    return
  fi
  "${ROOT_DIR}/hack/releases.sh" --bin kwokctl --version "${VERSION}" --platform "${GOOS}/${GOARCH}"
}

function build_kwok() {
  if [[ -f "${KWOK_CONTROLLER_BINARY}" ]]; then
    return
  fi
  "${ROOT_DIR}/hack/releases.sh" --bin kwok --version "${VERSION}" --platform "${GOOS}/${GOARCH}"
}

function build_image() {
  builder=${1:-"docker"}
  if ${builder} image inspect "${KWOK_CONTROLLER_IMAGE}" >/dev/null 2>&1; then
    return
  fi
  "${ROOT_DIR}/hack/releases.sh" --bin kwok --version "${KWOK_CONTROLLER_IMAGE##*:}" --platform "linux/${GOARCH}"
  "${ROOT_DIR}/images/kwok/build.sh" --image "${KWOK_CONTROLLER_IMAGE%%:*}" --version "${VERSION}" --builder "${builder}" --platform "linux/${GOARCH}"
}

function requirements() {
  install_kubectl
  install_buildx
  build_kwokctl
  build_image
}

function requirements_for_podman() {
  install_kubectl
  build_kwokctl
  build_image podman
}

function requirements_for_nerdctl() {
  install_kubectl
  build_kwokctl
  build_image nerdctl
}

function requirements_for_binary() {
  install_kubectl
  build_kwokctl
  build_kwok
}
