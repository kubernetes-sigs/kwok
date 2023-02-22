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

  KWOK_CONTROLLER_PORT=10247 KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --name "${name}" --timeout 10m --wait 10m --quiet-pull --config="${DIR}/port-forward.yaml"
  if [[ $? -ne 0 ]]; then
    echo "Error: Cluster ${name} creation failed"
    exit 1
  fi
}

function test_port_forward() {
  local name="${1}"
  local namespace="${2}"
  local target="${3}"
  local port="${4}"
  local pid
  local result
  kwokctl --name "${name}" kubectl -n "${namespace}" port-forward "${target}" 8888:"${port}" 2>&1 > /dev/null &
  pid=$!

  # allow some time for port forward to start
  sleep 5
  result=$(curl localhost:8888/healthz 2>/dev/null)
  kill -INT $pid
  if [[ ! "${result}" == "ok" ]]; then
    echo "Error: failed to port-forward to ${2}"
    return 1
  fi
  echo "${result}"
}

function test_port_forward_failed() {
    local name="${1}"
    local namespace="${2}"
    local target="${3}"
    local port="${4}"
    local pid
    local result
    kwokctl --name "${name}" kubectl -n "${namespace}" port-forward "${target}" 8888:"${port}" 2>&1 > /dev/null &
    pid=$!

    # allow some time for port forward to start
    sleep 5
    if curl localhost:8888/healthz 2>/dev/null; then
      echo "Error: port-forward to ${2} should fail"
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
  kwokctl --name "${name}" kubectl wait pod -A --all --for=condition=Ready --timeout=30s
  if [[ $? -ne 0 ]]; then
    echo "Error: fake-pod wait failed"
    echo kubectl get pod -A --all
    kubectl get pod -A --all
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
    echo "Testing port-forward on ${KWOK_RUNTIME} for ${release}"
    name="port-forward-cluster-${KWOK_RUNTIME}-${release//./-}"
    test_create_cluster "${release}" "${name}" || failed+=("create_cluster_${name}")
    test_apply_node_and_pod || failed+=("apply_node_and_pod")
    test_port_forward "${name}" other pod/fake-pod "8001" || failed+=("${name}_target_forward")
    test_port_forward "${name}" other pod/fake-pod "8002" || failed+=("${name}_command_forward")
    test_port_forward_failed "${name}" other pod/fake-pod "8003" || failed+=("${name}_failed")
    test_port_forward "${name}" default deploy/fake-pod "8004" || failed+=("${name}_cluster_default_forward")
    test_port_forward_failed "${name}" default deploy/fake-pod "8005" || failed+=("${name}_cluster_failed")
    test_delete_cluster "${release}" "${name}" || failed+=("delete_cluster_${name}")
  done

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "------------------------------"
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    echo kubectl get pod -A --all
    kubectl get pod -A --all
    exit 1
  fi
}

args "$@"

main
