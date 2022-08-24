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

ROOT="$(realpath "${DIR}/../..")"

KWOK_RUNTIME=""
KWOK_IMAGE="kwok"
KWOK_VERSION="test"

function args() {
  if [[ "${KWOK_RUNTIME}" == "" ]]; then
    if [[ "$(go env GOOS)" == "linux" ]]; then
      KWOK_RUNTIME=binary
    else
      KWOK_RUNTIME=docker
    fi
  fi
}

function main() {
  local platform
  local linux_platform
  platform="$(go env GOOS)/$(go env GOARCH)"
  "${ROOT}"/hack/releases.sh --bin kwokctl --platform "${platform}"
  if [[ "${KWOK_RUNTIME}" == "binary" ]]; then
    export KWOK_CONTROLLER_BINARY="${ROOT}/bin/${platform}/kwok"
    "${ROOT}"/hack/releases.sh --bin kwok --platform "${platform}"
  else
    export KWOK_CONTROLLER_IMAGE="${KWOK_IMAGE}:${KWOK_VERSION}"
    linux_platform="linux/$(go env GOARCH)"
    "${ROOT}"/hack/releases.sh --bin kwok --platform "${linux_platform}"
    "${ROOT}"/images/kwok/build.sh --image "${KWOK_IMAGE}" --version="${KWOK_VERSION}" --platform "${linux_platform}"
  fi

  echo Test workable

  PATH="${ROOT}/bin/${platform}/:${PATH}" "${DIR}/kwokctl_workable_test.sh" $(cat "${ROOT}"/supported_releases.txt) || exit 1

  echo Test benchmark

  PATH="${ROOT}/bin/${platform}/:${PATH}" "${DIR}/kwokctl_benchmark_test.sh" $(cat "${ROOT}"/supported_releases.txt | head -n 1) || exit 1
}

args "$@"

main
