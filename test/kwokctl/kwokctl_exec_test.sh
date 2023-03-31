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

function test_create_cluster() {
  local release="${1}"
  local name="${2}"

  KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --name "${name}" --timeout 10m --wait 10m --quiet-pull --config="${DIR}/exec.yaml"
  if [[ $? -ne 0 ]]; then
    echo "Error: Cluster ${name} creation failed"
    exit 1
  fi
}

function test_exec() {
  local name="${1}"
  local namespace="${2}"
  local target="${3}"
  local cmd="${4}"
  local want="${5}"
  local result
  result=$(kwokctl --name "${name}" kubectl -n "${namespace}" exec "${target}" -- "${cmd}")
  if [[ $? -ne 0 ]]; then
      echo "Error: exec failed"
      return 1
  fi

  if [[ ! "${result}" =~ "${want}" ]]; then
      echo "Error: exec result does not match"
      echo "  want: ${want}"
      echo "  got:  ${result}"
      return 1
  fi
}

function test_apply_node_and_pod() {
  kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-node.yaml"
  if [[ $? -ne 0 ]]; then
    echo "Error: fake-node apply failed"
    exit 1
  fi
  kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-pod-in-other-ns.yaml"
  if [[ $? -ne 0 ]]; then
    echo "Error: fake-pod apply failed"
    exit 1
  fi
  kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-deployment.yaml"
  if [[ $? -ne 0 ]]; then
    echo "Error: fake-deployment apply failed"
    exit 1
  fi
  kwokctl --name "${name}" kubectl wait pod -A --all --for=condition=Ready --timeout=60s
  if [[ $? -ne 0 ]]; then
    echo "Error: fake-pod wait failed"
    echo kwokctl --name "${name}" kubectl get pod -A --all
    kwokctl --name "${name}" kubectl get pod -A --all
    exit 1
  fi
}

function test_delete_cluster() {
  local release="${1}"
  local name="${2}"
  kwokctl delete cluster --name "${name}"
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing exec on ${KWOK_RUNTIME} for ${release}"
    name="exec-cluster-${KWOK_RUNTIME}-${release//./-}"
    test_create_cluster "${release}" "${name}" || failed+=("create_cluster_${name}")
    test_apply_node_and_pod || failed+=("apply_node_and_pod")
    test_exec "${name}" other pod/fake-pod "pwd" "/tmp" || failed+=("${name}_target_exec")
    test_exec "${name}" default deploy/fake-pod "env" "TEST_ENV=test"  || failed+=("${name}_cluster_default_exec")
    test_delete_cluster "${release}" "${name}" || failed+=("delete_cluster_${name}")
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
