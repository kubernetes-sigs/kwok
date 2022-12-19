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

  KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --name "${name}" --timeout 10m --wait 10m --quiet-pull
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

function get_snapshot_info() {
  local name="${1}"
  kwokctl --name "${name}" kubectl get pod | awk '{print $1, $2}'
  kwokctl --name "${name}" kubectl get node | awk '{print $1}'
}

function test_snapshot() {
  local release="${1}"
  local name="${2}"
  local empty_info
  local full_info
  local restore_empty_info
  local restore_full_info
  local empty_path="./snapshot-empty-${name}"
  local full_path="./snapshot-full-${name}"

  empty_info="$(get_snapshot_info "${name}")"

  kwokctl snapshot save --name "${name}" --path "${empty_path}"

  for ((i = 0; i < 30; i++)); do
    kubectl kustomize "${DIR}" | kwokctl --name "${name}" kubectl apply -f - && break
    sleep 1
  done

  for ((i = 0; i < 30; i++)); do
    full_info="$(get_snapshot_info "${name}")"
    if [[ "${full_info}" != "${empty_info}" && "${full_info}" =~ "default pod/" ]]; then
      break
    fi
    sleep 1
  done

  if [[ "${full_info}" == "${empty_info}" ]]; then
    echo "Error: Resource creation failed"
    return 1
  fi

  kwokctl snapshot save --name "${name}" --path "${full_path}"

  sleep 1
  kwokctl snapshot restore --name "${name}" --path "${empty_path}"
  for ((i = 0; i < 30; i++)); do
    restore_empty_info="$(get_snapshot_info "${name}")"
    if [[ "${empty_info}" == "${restore_empty_info}" ]]; then
      break
    fi
    sleep 1
  done

  if [[ "${empty_info}" != "${restore_empty_info}" ]]; then
    echo "Error: Empty snapshot restore failed"
    echo "Expected: ${empty_info}"
    echo "Actual: ${restore_empty_info}"
    return 1
  fi

  sleep 1

  kwokctl snapshot restore --name "${name}" --path "${full_path}"
  for ((i = 0; i < 30; i++)); do
    restore_full_info=$(get_snapshot_info "${name}")
    if [[ "${full_info}" == "${restore_full_info}" ]]; then
      break
    fi
    sleep 1
  done

  if [[ "${full_info}" != "${restore_full_info}" ]]; then
    echo "Error: Full snapshot restore failed"
    echo "Expected: ${full_info}"
    echo "Actual: ${restore_full_info}"
    return 1
  fi

  rm -rf "${empty_path}" "${full_path}"
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing snapshot on ${KWOK_RUNTIME} for ${release}"
    name="snapshot-cluster-${KWOK_RUNTIME}-${release//./-}"
    test_create_cluster "${release}" "${name}" || failed+=("create_cluster_${name}")
    test_snapshot "${release}" "${name}" || failed+=("snapshot_${name}")
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
