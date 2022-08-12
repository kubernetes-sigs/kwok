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
  local name="cluster-${release//./-}"
  local targets
  local i

  KWOK_KUBE_VERSION="${release}" kwokctl create cluster --name "${name}" --quiet-pull --prometheus-port 9090
  if [[ $? -ne 0 ]]; then
    echo "Cluster ${name} creation failed"
    exit 1
  fi

  for ((i = 0; i < 30; i++)); do
    kwokctl --name "${name}" kubectl apply -f - <<EOF
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
    kubernetes.io/hostname: fake-node
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok-controller
  name: fake-node
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fake-pod
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fake-pod
  template:
    metadata:
      labels:
        app: fake-pod
    spec:
      containers:
        - name: fake-pod
          image: fake
EOF
    if kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  echo kwokctl --name="${name}" kubectl config view --minify
  kwokctl --name="${name}" kubectl config view --minify

  if ! kwokctl --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
    echo kwokctl --name="${name}" logs kube-apiserver
    kwokctl --name="${name}" logs kube-apiserver
    echo
    echo kwokctl --name="${name}" logs kube-controller-manager
    kwokctl --name="${name}" logs kube-controller-manager
    echo
    echo kwokctl --name="${name}" logs kube-scheduler
    kwokctl --name="${name}" logs kube-scheduler
    echo
    echo kwokctl --name="${name}" logs kwok-controller
    kwokctl --name="${name}" logs kwok-controller
    echo
    return 1
  fi
}

function test_delelte_cluster() {
  local release="${1}"
  local name="cluster-${release//./-}"
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
    echo curl -s http://127.0.0.1:9090/api/v1/targets
    echo "${targets}"
    return 1
  fi
}

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    test_create_cluster "${release}" || failed+=("create_cluster_${release}")
    test_prometheus || failed+=("prometheus_${release}")
    test_delelte_cluster "${release}" || failed+=("delete_cluster_${release}")
  done

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

args "$@"

main
