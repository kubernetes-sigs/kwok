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

function child_timeout() {
  local to="${1}"
  shift
  "${@}" &
  local wp=$!
  local start=0
  while kill -0 "${wp}" 2>/dev/null; do
    if [[ "${start}" -ge "${to}" ]]; then
      kill "${wp}"
      echo "Error: Timeout ${to}s" >&2
      return 1
    fi
    ((start++))
    sleep 1
  done
  echo "Took ${start}s" >&2
}

function wait_resource() {
  local name="${1}"
  local reason="${2}"
  local want="${3}"
  local raw
  local got
  local all
  while true; do
    raw="$(kwokctl kubectl get --no-headers "${name}" 2>/dev/null)"
    got=$(echo "${raw}" | grep -c "${reason}")
    if [ "${got}" -eq "${want}" ]; then
      echo "${name} ${got} done"
      break
    else
      all=$(echo "${raw}" | wc -l)
      echo "${name} ${got}/${all} => ${want}"
    fi
    sleep 1
  done
}

function gen_pods() {
  local size="${1}"
  local node_name="${2}"
  for ((i = 0; i < "${size}"; i++)); do
    cat <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: fake-pod-${i}
  namespace: default
  labels:
    app: fake-pod
spec:
  containers:
  - name: fake-pod
    image: fake
  nodeName: ${node_name}
EOF
  done
}

function gen_nodes() {
  local size="${1}"
  local node_name="${2}"
  for ((i = 0; i < "${size}"; i++)); do
    cat <<EOF
---
apiVersion: v1
kind: Node
metadata:
  annotations:
    kwok.x-k8s.io/node: fake
    node.alpha.kubernetes.io/ttl: "0"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: ${node_name}-${i}
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok-controller
  name: ${node_name}-${i}
EOF
  done
}

function scale_create_pod() {
  local size="${1}"
  local node_name
  node_name="$(kwokctl kubectl get node -o jsonpath='{.items.*.metadata.name}' | tr ' ' '\n' | grep fake- | head -n 1)"
  gen_pods "${size}" "${node_name}" | kwokctl kubectl create -f - >/dev/null 2>&1 &
  wait_resource Pod Running "${size}"
}

function scale_delete_pod() {
  local size="${1}"
  kwokctl kubectl delete pod -l app=fake-pod --grace-period 1 >/dev/null 2>&1 &
  wait_resource Pod fake-pod- "${size}"
}

function scale_create_node() {
  local size="${1}"
  gen_nodes "${size}" "fake-node" | kwokctl kubectl create -f - >/dev/null 2>&1 &
  wait_resource Node Ready "${size}"
}

function create_cluster() {
  KWOK_KUBE_VERSION="${KWOK_KUBE_VERSION}" kwokctl create cluster --quiet-pull || {
    echo "Error: Failed to create cluster" >&2
    exit 1
  }
}

function delete_cluster() {
  kwokctl delete cluster
}

function main() {
  local failed=()
  local name

  echo "------------------------------"
  echo "Benchmarking on ${KWOK_RUNTIME}"
  name="benchmark-${KWOK_RUNTIME}"

  create_cluster
  scale_create_node 1
  child_timeout 120 scale_create_pod 10000 || failed+=("scale_create_pod_timeout_${name}")
  child_timeout 120 scale_delete_pod 0 || failed+=("scale_delete_pod_timeout_${name}")
  delete_cluster

  create_cluster
  child_timeout 180 scale_create_node 10000 || failed+=("scale_create_node_timeout_${name}")
  delete_cluster

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
