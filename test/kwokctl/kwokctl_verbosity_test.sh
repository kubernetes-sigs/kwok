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

  KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --name "${name}" --timeout 30m --wait 30m --quiet-pull --prometheus-port 9090

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

function check_kind() {
  local name="${1}"

  for pod in etcd kube-apiserver kube-controller-manager kube-scheduler; do
    kwokctl --name="${name}" kubectl -n kube-system get pod "${pod}-kwok-${name}-control-plane" -oyaml | grep -q -- "--v=4\|--log-level=debug" || return 1
  done
  kwokctl --name="${name}" kubectl -n kube-system get pod prometheus -oyaml | grep -q -- "--log.level=debug" || return 1
}

function check_docker() {
  local name="${1}"

  for component in etcd kube-apiserver kube-controller-manager kube-scheduler prometheus; do
    docker inspect kwok-"${name}"-"${component}" | grep -q -- "--v=4\|--log-level=debug\|--log.level=debug" || return 1
  done
}

function check_nerdctl() {
  local name="${1}"

  for component in etcd kube-apiserver kube-controller-manager kube-scheduler prometheus; do
    nerdctl inspect kwok-"${name}"-"${component}" | grep -q -- "--v=4\|--log-level=debug\|--log.level=debug" || return 1
  done
}

function check_binary() {
  local name="${1}"

  for component in etcd kube-apiserver kube-controller-manager kube-scheduler prometheus; do
    pgrep -a -f "${component}" | grep -q -- "--v=4\|--log-level=debug\|--log.level=debug" || return 1
  done
}

function test_verbosity() {
  local runtime="${KWOK_RUNTIME}"
  local release="${1}"
  local name="${2}"

  if [[ "${runtime}" == "kind" ]]; then
    check_kind "${name}"
  elif [[ "${runtime}" == "docker" ]]; then
    check_docker "${name}"
  elif [[ "${runtime}" == "nerdctl" ]]; then
    check_nerdctl "${name}"
  elif [[ "${runtime}" == "binary" ]]; then
    check_binary "${name}"
  fi

  if [[ $? -ne 0 ]]; then
    echo "Error: '-v' flag not exsits in kube-apiserver"
    return 1
  fi
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing verbosity on ${KWOK_RUNTIME} for ${release}"
    name="verbosity-cluster-${KWOK_RUNTIME}-${release//./-}"
    test_create_cluster "${release}" "${name}" || failed+=("create_cluster_${name}")
    test_verbosity "${release}" "${name}" || failed+=("verbosity_${name}")
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
