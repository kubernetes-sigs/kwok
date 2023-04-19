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

function test_check_clusters() {
  local names=("$@")
  local want=()
  for name in "${names[@]}"; do
    want+=("${name}")
  done
  if [[ "${#names[@]}" -ne "${#want[@]}" ]]; then
    echo "Error: Cluster list size does not match"
    echo "Expected: ${#names[@]}"
    echo "Actual: ${#want[@]}"
    return 1
  fi
  for name in "${names[@]}"; do
    local found=0
    for w in "${want[@]}"; do
      if [[ "${name}" == "${w}" ]]; then
        found=1
        break
      fi
    done
    if [[ "${found}" -ne 1 ]]; then
      echo "Error: Cluster ${name} not found"
      return 1
    fi
  done
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing multi-cluster on ${KWOK_RUNTIME} for ${release}"
    name1="multi-cluster-1-cluster-${KWOK_RUNTIME}-${release//./-}"
    name2="multi-cluster-2-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name1}" "${release}"
    test_check_clusters "${name1}" || failed+=("check_cluster_${name1}")
    create_cluster "${name2}" "${release}"
    test_check_clusters "${name1}" "${name2}" || failed+=("check_cluster_${name1}_${name2}")
    delete_cluster "${name1}"
    test_check_clusters "${name2}" || failed+=("check_cluster_${name2}")
    delete_cluster "${name2}"
    test_check_clusters || failed+=("check_cluster")
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
