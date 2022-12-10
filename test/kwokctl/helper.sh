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

function test_all() {
  local runtime="${1}"
  local cases="${2}"
  local releases=("${@:3}")

  echo "Test ${cases} on ${runtime} for ${releases[*]}"
  KWOK_RUNTIME="${runtime}" PATH="${DIR}/bin:${PATH}" "${DIR}/kwokctl_${cases}_test.sh" "${releases[@]}"
}

export KWOK_CONTROLLER_BINARY="${DIR}/bin/kwok"
export KWOK_CONTROLLER_IMAGE=local/kwok:test

# Test only the latest releases of Kubernetes
LAST_RELEASE_SIZE="${LAST_RELEASE_SIZE:-6}"

function supported_releases() {
  cat "${ROOT_DIR}/supported_releases.txt" | head -n "${LAST_RELEASE_SIZE}"
}

function build_kwokctl() {
  if [[ -f "${DIR}/bin/kwokctl" ]]; then
    return
  fi
  go build -o "${DIR}/bin/kwokctl" "${ROOT_DIR}/cmd/kwokctl"
}

function build_kwok() {
  if [[ -f "${KWOK_CONTROLLER_BINARY}" ]]; then
    return
  fi
  go build -o "${KWOK_CONTROLLER_BINARY}" "${ROOT_DIR}/cmd/kwok"
}

function build_image() {
  if docker image inspect "${KWOK_CONTROLLER_IMAGE}" >/dev/null 2>&1; then
    return
  fi
  "${ROOT_DIR}/hack/releases.sh" --bin kwok
  "${ROOT_DIR}/images/kwok/build.sh" --image "${KWOK_CONTROLLER_IMAGE%%:*}" --version="${KWOK_CONTROLLER_IMAGE##*:}"
}

function build_image_for_nerdctl() {
  build_image
  mkdir "tmp"
  docker save -o "tmp/kwok.tar" "${KWOK_CONTROLLER_IMAGE}"
  nerdctl load -i "tmp/kwok.tar"
}
