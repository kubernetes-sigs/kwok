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

  KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --name "${name}" --timeout 10m --wait 10m --quiet-pull --prometheus-port 9090
  if [[ $? -ne 0 ]]; then
    echo "Error: Cluster ${name} creation failed"
    exit 1
  fi
}

function test_delete_cluster() {
  local release="${1}"
  local name="${2}"
  kwokctl delete cluster --name "${name}"
}

function test_prometheus() {
  local targets
  for ((i = 0; i < 30; i++)); do
    targets="$(curl -s http://127.0.0.1:9090/api/v1/targets)"
    if [[ "$(echo "${targets}" | grep -o '"health":"up"' | wc -l)" -ge 6 ]]; then
      break
    fi
    sleep 1
  done

  if ! [[ "$(echo "${targets}" | grep -o '"health":"up"' | wc -l)" -ge 6 ]]; then
    echo "Error: metrics is not health"
    echo curl -s http://127.0.0.1:9090/api/v1/targets
    echo "${targets}"
    return 1
  fi
}

function get_resource_info() {
  local name="${1}"
  kwokctl --name "${name}" kubectl get pod --all-namespaces | awk '{print $1, $2}'
  kwokctl --name "${name}" kubectl get node | awk '{print $1}'
}

function test_restart() {
  local release="${1}"
  local name="${2}"
  local expect_info
  local actual_info

  test_prometheus
  if [[ $? -ne 0 ]]; then
    echo "Error: cluster ${name} not ready"
    return 1
  fi
  kubectl kustomize "${DIR}" | kwokctl --name "${name}" kubectl apply -f -
  sleep 15
  expect_info="$(get_resource_info "${name}")"

  echo  kwokctl --name "${name}" stop cluster
  kwokctl --name "${name}" stop cluster
  if [[ $? -eq 0 ]]; then
      echo "Cluster ${name} stopped successfully."
  else
      echo "Error: cluster ${name} stop error"
      return 1
  fi
  sleep 5

  kwokctl --name "${name}" kubectl get no
  if [[ $? -eq 0 ]]; then
    echo "Error: cluster ${name} do not stop"
    return 1
  fi

  echo kwokctl --name "${name}" start cluster --wait 10m --timeout 10m
  kwokctl --name "${name}" start cluster --wait 10m --timeout 10m
  if [[ $? -eq 0 ]]; then
      echo "Cluster ${name} started successfully."
  else
      echo "Error: cluster ${name} start error"
      return 1
  fi
  test_prometheus
  if [[ $? -ne 0 ]]; then
    echo "Error: cluster ${name} not restart"
    return 1
  fi

  actual_info="$(get_resource_info "${name}")"
  if [[ "${expect_info}" != "${actual_info}" ]]; then
    echo "Error: Cluster ${name} start failed"
    echo "Expected: ${expect_info}"
    echo "Actual: ${actual_info}"
    return 1
  fi
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing restart on ${KWOK_RUNTIME} for ${release}"
    name="restart-cluster-${KWOK_RUNTIME}-${release//./-}"
    test_create_cluster "${release}" "${name}" || failed+=("create_cluster_${name}")
    test_restart "${release}" "${name}" || failed+=("restart_cluster_${name}")
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
