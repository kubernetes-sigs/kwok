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

KWOK_RUNTIME=""

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
  go build -o "${DIR}/bin/kwokctl" "${DIR}"/../../cmd/kwokctl

  if [[ "${KWOK_RUNTIME}" == "binary" ]]; then
    export KWOK_CONTROLLER_BINARY="${DIR}/bin/kwok"
    go build -o "${KWOK_CONTROLLER_BINARY}" "${DIR}"/../../cmd/controller
  else
    export KWOK_CONTROLLER_IMAGE=kwok:test
    "${DIR}"/../../images/kwok/build.sh --tag "${KWOK_CONTROLLER_IMAGE}"
  fi

  echo Test workable

  PATH="${DIR}/bin:${PATH}" "${DIR}/kwokctl_workable_test.sh" $(cat "${DIR}"/../../supported_releases.txt) || exit 1

  echo Test benchmark

  PATH="${DIR}/bin:${PATH}" "${DIR}/kwokctl_benchmark_test.sh" $(cat "${DIR}"/../../supported_releases.txt | head -n 1) || exit 1
}

args "$@"

main
