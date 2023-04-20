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

function test_scheduler() {
  local name="${1}"

  for ((i = 0; i < 120; i++)); do
    kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-node.yaml"
    if kwokctl --name="${name}" kubectl get node | grep Ready >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  for ((i = 0; i < 120; i++)); do
    kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-scheduler-deployment.yaml"
    if kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
    echo "Error: cluster not ready"
    show_all
    return 1
  fi
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing scheduler on ${KWOK_RUNTIME} for ${release}"
    name="scheduler-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --kube-scheduler-config "${DIR}/scheduler-config.yaml"
    test_scheduler "${name}" || failed+=("scheduler_${name}")
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
