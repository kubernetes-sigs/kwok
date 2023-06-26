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

function test_apply_node_and_pod() {
  local name="${1}"
  if ! kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-node.yaml"; then
    echo "Error: fake-node apply failed"
    return 1
  fi
  if ! retry 120 kwokctl --name "${name}" kubectl apply -f "${DIR}/fake-pod-in-other-ns.yaml"; then
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

function metric_value() {
  local result="${1}"
  local metric_name="${2}"
  echo "${result}" | grep "^${metric_name}" | cut -d " " -f 2
}

function test_node_metrics() {
  local result
  local found_metric=0
  for ((i = 0; i < 15; i++)); do
    result="$(curl http://localhost:10250/metrics/nodes/fake-node)"
    if [[ "$(metric_value "${result}" 'kubelet_node_name{node="fake-node"}')" -eq 1 ]]; then
      found_metric=1
      break
    fi
    sleep 1
  done

  if [[ "${found_metric}" -ne 1 ]]; then
    echo "Error: kubelet_node_name of fake-node is not equal to 1"
    return 1
  fi

  found_metric=0
  for ((i = 0; i < 15; i++)); do
    result="$(curl http://localhost:10250/metrics/nodes/fake-node)"
    if [[ "$(metric_value "${result}" "kubelet_started_containers_total")" -ne 1 ]]; then
      found_metric=1
      break
    fi
    sleep 1
  done

  if [[ "${found_metric}" -ne 1 ]]; then
    echo "Error: started containers total is not equal to 1"
    return 1
  fi

  if [[ -z "$(metric_value "${result}" "kubelet_pleg_relist_duration_seconds_count")" ]]; then
    echo "Error: kubelet_pleg_relist_duration_seconds_count not set"
    return 1
  fi

  if [[ -z "$(metric_value "${result}" "container_cpu_usage_seconds_total")" ]]; then
    echo "Error: container_cpu_usage_seconds_total not set"
    return 1
  fi

  if [[ -z "$(metric_value "${result}" "pod_cpu_usage_seconds_total")" ]]; then
    echo "Error: pod_cpu_usage_seconds_total not set"
    return 1
  fi
}

function main() {
  local failed=()

  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing metrics on ${KWOK_RUNTIME} for ${release}"
    name="metric-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --config "${DIR}/metric.yaml" --prometheus-port "9090" --controller-port "10250"
    test_apply_node_and_pod "${name}" || failed+=("apply_node_and_pod")
    test_node_metrics "${name}" || failed+=("test_node_metrics")
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
