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

ROOT_DIR="$(realpath "${DIR}/../..")"

BASH_IMAGE=registry.k8s.io/build-image/distroless-iptables:v0.5.1
CLUSTER_NAME=kwok-test
KWOK_IMAGE="kwok-with-cni"
KWOK_VERSION="test"

export PATH="${ROOT_DIR}/bin:${PATH}"

function start_cluster() {
  local linux_platform
  linux_platform="linux/$(go env GOARCH)"
  "${ROOT_DIR}"/hack/releases.sh --bin kwok --platform "${linux_platform}"
  "${ROOT_DIR}"/images/kwok/build.sh --base-image ${BASH_IMAGE} --image "${KWOK_IMAGE}" --version="${KWOK_VERSION}" --platform "${linux_platform}"

  kind create cluster --name="${CLUSTER_NAME}"

  kind load docker-image --name="${CLUSTER_NAME}" "${KWOK_IMAGE}:${KWOK_VERSION}"

  kubectl kustomize "${DIR}" | kubectl apply -f -
  kubectl kustomize "${ROOT_DIR}/stages" | kubectl apply -f -
}

# Check for normal heartbeat
function test_node_ready() {
  for ((i = 0; i < 300; i++)); do
    if [[ ! "$(kubectl get node fake-node)" =~ "Ready" ]]; then
      echo "Waiting for fake-node to be ready..."
      sleep 1
    else
      break
    fi
  done

  if [[ ! "$(kubectl get node fake-node)" =~ "Ready" ]]; then
    echo "Error: fake-node is not ready"
    kubectl get node fake-node
    return 1
  fi
}

# Check for the Pod is running
function test_pod_running() {
  for ((i = 0; i < 300; i++)); do
    if [[ ! "$(kubectl get pod -o wide | grep -c Running)" -eq 5 ]]; then
      echo "Waiting all pods to be running..."
      sleep 1
    else
      break
    fi
  done

  if [[ ! "$(kubectl get pod -o wide | grep -c Running)" -eq 5 ]]; then
    echo "Error: Not all pods are running"
    kubectl get pod -o wide
    return 1
  fi
}

# Cleanup
function cleanup() {
  kind delete cluster --name="${CLUSTER_NAME}"
}

function main() {
  failed=()
  start_cluster || {
    echo "Error: Failed to start cluster"
    exit 1
  }
  trap cleanup EXIT

  test_node_ready || failed+=("node_ready")
  test_pod_running || failed+=("pod_running")

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

main
