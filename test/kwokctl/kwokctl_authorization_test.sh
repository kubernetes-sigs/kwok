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

source "${DIR}/suite.sh"

RELEASES=()

function usage() {
  echo "Usage: $0 <kube-version...>"
  echo "  <kube-version> is the version of kubernetes to test against."
}

function args() {
  if [[ $# -eq 0 ]]; then
    usage
    exit 1
  fi
  while [[ $# -gt 0 ]]; do
    RELEASES+=("${1}")
    shift
  done
}

function test_authorization() {
  local name="${1}"
  local resource
  resource="$(kwokctl --name "${name}" kubectl get role,rolebinding,clusterrole,clusterrolebinding -A)"
  if [[ "${resource}" == "" ]]; then
    echo "Error: role,rolebinding,clusterrole,clusterrolebinding is empty"
    return 1
  fi
  echo "${resource}"
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing authorization on ${KWOK_RUNTIME} for ${release}"
    name="authorization-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --kube-authorization
    test_authorization "${name}" || failed+=("authorization_${name}")
    delete_cluster "${name}"
  done

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "------------------------------"
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

args "$@"

main
