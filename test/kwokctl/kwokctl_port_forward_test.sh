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

function test_port_forward() {
  local name="${1}"
  local namespace="${2}"
  local target="${3}"
  local port="${4}"
  local pid
  local result
  kwokctl --name "${name}" kubectl -n "${namespace}" port-forward "${target}" 8888:"${port}" >/dev/null 2>&1 &
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
  kwokctl --name "${name}" kubectl -n "${namespace}" port-forward "${target}" 8888:"${port}" >/dev/null 2>&1 &
  pid=$!

  # allow some time for port forward to start
  sleep 5
  if curl localhost:8888/healthz 2>/dev/null; then
    echo "Error: port-forward to ${2} should fail"
    return 1
  fi
}

function test_apply_node_and_pod() {
  local name="${1}"
  if ! kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-node.yaml"; then
    echo "Error: fake-node apply failed"
    return 1
  fi
<<<<<<< HEAD
  if ! retry 120 kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-pod-in-other-ns.yaml"; then
=======
  kwokctl --name "${name}" kubectl create ns other
  if [[ $? -ne 0 ]]; then
    echo "Error: other-namespace create failed"
    return 1
  fi
  for ((i = 0; i < 120; i++)); do
    kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-pod.yaml"
    if [[ $? -eq 0 ]]; then
      break
    fi
    sleep 1
  done
  if [[ $? -ne 0 ]]; then
>>>>>>> c3114f7 (fix exec)
    echo "Error: fake-pod apply failed"
    return 1
  fi
  if ! kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-deployment.yaml"; then
    echo "Error: fake-deployment apply failed"
    return 1
  fi
  if ! kwokctl --name "${name}" kubectl wait pod -A --all --for=condition=Ready --timeout=60s; then
    echo "Error: fake-pod wait failed"
    echo kwokctl --name "${name}" kubectl get pod -A --all
    kwokctl --name "${name}" kubectl get pod -A --all
    return 1
  fi
}

export KWOK_CONTROLLER_PORT=10247

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing port-forward on ${KWOK_RUNTIME} for ${release}"
    name="port-forward-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --config "${DIR}/port-forward.yaml"
    test_apply_node_and_pod "${name}" || failed+=("apply_node_and_pod")
    test_port_forward "${name}" other pod/fake-pod "8001" || failed+=("${name}_target_forward")
    test_port_forward "${name}" other pod/fake-pod "8002" || failed+=("${name}_command_forward")
    test_port_forward_failed "${name}" other pod/fake-pod "8003" || failed+=("${name}_failed")
    test_port_forward "${name}" default deploy/fake-pod "8004" || failed+=("${name}_cluster_default_forward")
    test_port_forward_failed "${name}" default deploy/fake-pod "8005" || failed+=("${name}_cluster_failed")
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
