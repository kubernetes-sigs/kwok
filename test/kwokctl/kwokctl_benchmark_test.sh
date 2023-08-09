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

KWOK_KUBE_VERSION=""

function usage() {
  echo "Usage: ${0} <kube-version>"
  echo "  <kube-version> is the version of kubernetes to test against."
}

function args() {
  if [[ $# -ne 1 || "${1}" == "" ]]; then
    usage
    exit 1
  fi

  KWOK_KUBE_VERSION="${1}"
}

function wait_resource() {
  local name="${1}"
  local resource="${2}"
  local reason="${3}"
  local want="${4}"
  local raw
  local got
  local all
  while true; do
    raw="$(kwokctl --name "${name}" kubectl get --no-headers "${resource}" | grep "fake-" 2>/dev/null)"
    got=$(echo "${raw}" | grep -c "${reason}")
    if [[ "${got}" == "${want}" ]]; then
      echo "${resource} ${got} done"
      break
    else
      all=$(echo "${raw}" | wc -l)
      echo "${resource} ${got}/${all} => ${want}"
    fi
    sleep 1
  done
}

function scale_create_pod() {
  local name="${1}"
  local size="${2}"
  local node_name
  node_name="$(kwokctl --name "${name}" kubectl get node -o jsonpath='{.items.*.metadata.name}' | tr ' ' '\n' | grep fake- | head -n 1)"
  kwokctl --name "${name}" scale pod fake-pod --replicas "${size}" --param ".nodeName=\"${node_name}\"" >/dev/null &
  wait_resource "${name}" Pod Running "${size}"
}

function scale_delete_pod() {
  local name="${1}"
  local size="${2}"
  kwokctl --name "${name}" scale pod fake-pod --replicas "${size}" >/dev/null &
  wait_resource "${name}" Pod fake-pod- "${size}"
}

function scale_create_node() {
  local name="${1}"
  local size="${2}"
  kwokctl --name "${name}" scale node fake-node --replicas "${size}" >/dev/null &
  wait_resource "${name}" Node Ready "${size}"
}

function main() {
  local failed=()
  local name
  local release="${KWOK_KUBE_VERSION}"

  echo "------------------------------"
  echo "Benchmarking on ${KWOK_RUNTIME}"
  name="benchmark-${KWOK_RUNTIME}"

  create_cluster "${name}" "${release}" --disable-qps-limits
  child_timeout 120 scale_create_node "${name}" 1000 || failed+=("scale_create_node_timeout_${name}")
  child_timeout 120 scale_create_pod "${name}" 1000 || failed+=("scale_create_pod_timeout_${name}")
  child_timeout 120 scale_delete_pod "${name}" 0 || failed+=("scale_delete_pod_timeout_${name}")
  delete_cluster "${name}"

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
