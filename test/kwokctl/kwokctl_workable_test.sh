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

function test_workable() {
  local name="${1}"
  local targets
  local current_context
  local i

  current_context="$(kubectl config current-context)"
  if [[ "${current_context}" != "kwok-${name}" ]]; then
    echo "Error: Current context is ${current_context}, expected kwok-${name}"
    return 1
  fi

  for ((i = 0; i < 120; i++)); do
    kubectl kustomize "${DIR}" | kwokctl --name "${name}" kubectl apply -f - && break
    sleep 1
  done

  for ((i = 0; i < 120; i++)); do
    kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1 && break
    sleep 1
  done

  echo kwokctl --name="${name}" kubectl config view --minify
  kwokctl --name="${name}" kubectl config view --minify

  if ! kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
    echo "Error: cluster not ready"
    show_all
    return 1
  fi

  if ! kwokctl --name="${name}" etcdctl get /registry/namespaces/default --keys-only | grep default >/dev/null 2>&1; then
    echo "Error: Failed to get namespace(default) by kwokctl etcdctl in cluster ${name}"
    show_all
    return 1
  fi
}

function test_prometheus() {
  local targets
  for ((i = 0; i < 120; i++)); do
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

function test_kwok_controller_port() {
  local result
  result="$(curl -s http://127.0.0.1:10247/healthz)"
  if [[ ! $result == "ok" ]]; then
    echo "Error: controller healthz is not ok"
    echo curl -s http://127.0.0.1:10247/healthz
    echo "${result}"
    return 1
  fi
}

function test_etcd_port() {
  local result
  result=$(curl -s http://127.0.0.1:2400/health)

  if [[ "$(echo "${result}" | grep -o '"health"' | wc -c)" = 0 ]]; then
    echo "Error: etcd connection"
    echo curl -s http://127.0.0.1:2400/health
    echo "${result}"
    return 1
  fi
}

function test_kube_scheduler_port() {
  local result

  local version="${1}"
  local minor="${version#*.}"
  minor="${minor%.*}"

  local proto="https"
  if [[ $minor -lt 13 ]]; then
    proto="http"
  fi

  result=$(curl -s -k "${proto}://127.0.0.1:10250/healthz")

  if [[ "${result}" != "ok" ]]; then
    echo "Error: kube scheduler connection"
    echo "curl -s ${proto}://127.0.0.1:10250/healthz"
    echo "${result}"
    return 1
  fi
}

function test_kube_controller_manager_port() {
  local result

  local version="${1}"
  local minor="${version#*.}"
  minor="${minor%.*}"

  local proto="https"
  if [[ $minor -lt 13 ]]; then
    proto="http"
  fi

  result=$(curl -s -k "${proto}://127.0.0.1:10260/healthz")

  if [[ "${result}" != "ok" ]]; then
    echo "Error: kube controller manager connection"
    echo "curl -s ${proto}://127.0.0.1:10260/healthz"
    echo "${result}"
    return 1
  fi
}

function main() {
  local failed=()
  local name
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing workable on ${KWOK_RUNTIME} for ${release}"
    name="cluster-${KWOK_RUNTIME}-${release//./-}"
    local minor="${release#*.}"
    minor="${minor%.*}"

    nerdctl compose --help
    nerdctl --version
    lima --version

    if [[ $minor -lt 13 ]]; then
      echo "no secure port"
      create_cluster "${name}" "${release}" -v=debug --prometheus-port 9090 --controller-port 10247 --etcd-port=2400 --kube-scheduler-port=10250 --kube-controller-manager-port=10260
    else
      echo "secure port"
      create_cluster "${name}" "${release}" -v=debug --secure-port --kube-admission --kube-authorization --prometheus-port 9090 --controller-port 10247 --etcd-port=2400 --kube-scheduler-port=10250 --kube-controller-manager-port=10260
    fi
    test_workable "${name}" || failed+=("workable_${name}")
    if [[ "${KWOK_RUNTIME}" != "kind" && "${KWOK_RUNTIME}" != "kind-podman" ]]; then
      test_kube_controller_manager_port "${release}" || failed+=("kube_controller_manager_port_${name}")
      test_kube_scheduler_port "${release}" || failed+=("kube_scheduler_port_${name}")
      test_etcd_port || failed+=("etcd_port_${name}")
    fi
    test_prometheus || failed+=("prometheus_${name}")
    test_kwok_controller_port || failed+=("kwok_controller_port_${name}")
    delete_cluster "${name}"
  done
  echo "------------------------------"

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
