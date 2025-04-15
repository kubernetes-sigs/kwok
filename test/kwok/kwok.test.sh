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

CLUSTER_NAME=kwok-test
KWOK_IMAGE="kwok"
KWOK_VERSION="test"

export PATH="${ROOT_DIR}/bin:${PATH}"

function start_cluster() {
  local linux_platform
  linux_platform="linux/$(go env GOARCH)"
  "${ROOT_DIR}"/hack/releases.sh --bin kwok --platform "${linux_platform}"
  "${ROOT_DIR}"/images/kwok/build.sh --image "${KWOK_IMAGE}" --version="${KWOK_VERSION}" --platform "${linux_platform}"

  kind create cluster --name="${CLUSTER_NAME}"

  kind load docker-image --name="${CLUSTER_NAME}" "${KWOK_IMAGE}:${KWOK_VERSION}"

  kubectl kustomize "${DIR}" | kubectl apply -f -
  kubectl kustomize "${ROOT_DIR}/kustomize/stage/fast" | kubectl apply -f -
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

# Check pod status
function test_check_pod_status() {
  _jq() {
    echo "${row}" | base64 --decode | jq -r "${1}"
  }
  for row in $(kubectl get po -ojson | jq -r '.items[] | @base64'); do
    pod_name=$(_jq '.metadata.name')
    node_name=$(_jq '.spec.nodeName')
    host_network=$(_jq '.spec.hostNetwork')
    host_ip=$(_jq '.status.hostIP')
    pod_ip=$(_jq '.status.podIP')
    node_ip=$(kubectl get nodes "${node_name}" -ojson | jq -r '.status.addresses[] | select( .type == "InternalIP" ) | .address')
    if [[ "${host_ip}" != "${node_ip}" ]]; then
      echo "Error: ${pod_name} Pod: HostIP (${host_ip}) is not equal to the IP on its Node (${node_ip})"
      return 1
    fi
    if [[ $host_network == "true" ]]; then
      if [[ "${pod_ip}" != "${node_ip}" ]]; then
        echo "Error: ${pod_name} Pod with hostNetwork=${host_network}: PodIP (${pod_ip}) and HostIP (${host_ip}) are not equal"
        return 1
      fi
    fi
  done
  echo "Pod's status checked"
}

# Check for the Pod is running
function test_pod_running() {
  for ((i = 0; i < 300; i++)); do
    if [[ ! "$(kubectl get pod | grep -c Running)" -eq 5 ]]; then
      echo "Waiting all pods to be running..."
      sleep 1
    else
      break
    fi
  done

  if [[ ! "$(kubectl get pod | grep -c Running)" -eq 5 ]]; then
    echo "Error: Not all pods are running"
    kubectl get pod -o wide
    return 1
  fi
}

# Check for the status of the Node is modified by Kubectl
function test_modify_node_status() {
  kubectl annotate node fake-node --overwrite kwok.x-k8s.io/status=custom
  kubectl patch node fake-node --subresource=status -p '{"status":{"nodeInfo":{"kubeletVersion":"fake-new"}}}'

  sleep 2

  if [[ ! "$(kubectl get node fake-node)" =~ "fake-new" ]]; then
    echo "Error: fake-node is not updated"
    kubectl get node fake-node
    return 1
  fi
  kubectl annotate node fake-node --overwrite kwok.x-k8s.io/status-
}

# Check for the status of the Pod is modified by Kubectl
function test_modify_pod_status() {
  local first_pod
  first_pod="$(kubectl get pod | grep Running | head -n 1 | awk '{print $1}')"

  kubectl annotate pod "${first_pod}" --overwrite kwok.x-k8s.io/status=custom
  kubectl patch pod "${first_pod}" --subresource=status -p '{"status":{"podIP":"192.168.0.1"}}'

  sleep 2

  if [[ ! "$(kubectl get pod "${first_pod}" -o wide)" == *"192.168.0.1"* ]]; then
    echo "Error: fake-pod is not updated"
    kubectl get pod "${first_pod}" -o wide
    return 1
  fi

  kubectl annotate pod "${first_pod}" --overwrite kwok.x-k8s.io/status-
}

function test_check_node_lease_transitions() {
  local want="${1}"
  local node_leases_transitions
  node_leases_transitions="$(kubectl get leases fake-node -n kube-node-lease -ojson | jq -r '.spec.leaseTransitions // 0')"
  if [[ "${node_leases_transitions}" != "${want}" ]]; then
    echo "Error: fake-node lease transitions is not ${want}, got ${node_leases_transitions}"
    return 1
  fi
}

function recreate_kwok() {
  kubectl scale deployment/kwok-controller -n kube-system --replicas=0
  kubectl wait --for=delete pod -l app=kwok-controller -n kube-system --timeout=60s

  kubectl scale deployment/kwok-controller -n kube-system --replicas=2
}

function recreate_pods() {
  kubectl delete pod --all -n default
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
  test_check_pod_status || failed+=("check_pod_status")

  test_modify_node_status || failed+=("modify_node_status")
  test_modify_pod_status || failed+=("modify_pod_status")
  test_check_node_lease_transitions 0 || failed+=("check_node_lease_transitions")

  recreate_kwok || failed+=("recreate_kwok")
  recreate_pods || failed+=("recreate_pods")
  test_node_ready || failed+=("node_ready_again")
  test_pod_running || failed+=("pod_running_again")
  test_check_pod_status || failed+=("check_pod_status_again")

  sleep 45

  test_check_node_lease_transitions 1 || failed+=("check_node_lease_transitions_again")

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

main
