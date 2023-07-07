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

function test_exec() {
  local cmds=()
  local name="${1}"
  local namespace="${2}"
  local target="${3}"
  local cmd="${4}"
  local want="${5}"
  local result
  mapfile -t cmds < <(echo "${cmd}" | tr " " "\n")
  for ((i = 0; i < 10; i++)); do
    result=$(kwokctl --name "${name}" kubectl -n "${namespace}" exec -i "${target}" -- "${cmds[@]}" || :)
    if [[ "${result}" == *"${want}"* ]]; then
      break
    fi
    sleep 1
  done

  if [[ ! "${result}" == *"${want}"* ]]; then
    echo "Error: exec result does not match"
    echo "  want: ${want}"
    echo "  got:  ${result}"
    return 1
  fi
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

function main() {
  local failed=()
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing exec on ${KWOK_RUNTIME} for ${release}"
    name="exec-cluster-${KWOK_RUNTIME}-${release//./-}"
    if [[ "${KWOK_RUNTIME}" != "binary" && "${KWOK_RUNTIME}" != "kind" && "${KWOK_RUNTIME}" != "kind-podman" ]]; then
      yaml="${DIR}/exec-security-context.yaml"
    else
      yaml="${DIR}/exec.yaml"
    fi
    create_cluster "${name}" "${release}" --config - <<EOF
apiVersion: config.kwok.x-k8s.io/v1alpha1
kind: KwokConfiguration
options:
  enableCRDs:
  - ClusterExec
  - Exec
EOF
    if [[ "${KWOK_RUNTIME}" != "binary" && "${KWOK_RUNTIME}" != "kind" && "${KWOK_RUNTIME}" != "kind-podman" ]]; then
      create_user "${KWOK_RUNTIME}" "${name}" "kwok-controller" 1001 "test" 1002 "test" "/home/test" "/bin/sh"
    fi
    test_apply_node_and_pod "${name}" || failed+=("apply_node_and_pod")
    kwokctl --name "${name}" kubectl apply -f "${yaml}"
    test_exec "${name}" other pod/fake-pod "pwd" "/tmp" || failed+=("${name}_target_exec")
    test_exec "${name}" default deploy/fake-pod "env" "TEST_ENV=test" || failed+=("${name}_cluster_default_exec")
    if [[ "${KWOK_RUNTIME}" != "binary" && "${KWOK_RUNTIME}" != "kind" && "${KWOK_RUNTIME}" != "kind-podman" ]]; then
      test_exec "${name}" default deploy/fake-pod "id -u" "1001" || failed+=("${name}_cluster_default_exec")
      test_exec "${name}" default deploy/fake-pod "id -g" "1002" || failed+=("${name}_cluster_default_exec")
    fi
    delete_cluster "${name}"

    name="crd-exec-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --config - <<EOF
apiVersion: config.kwok.x-k8s.io/v1alpha1
kind: KwokConfiguration
options:
  enableCRDs:
  - ClusterExec
  - Exec
EOF
    test_apply_node_and_pod "${name}" || failed+=("apply_node_and_pod")
    kwokctl --name "${name}" kubectl apply -f "${DIR}/exec.yaml"
    test_exec "${name}" other pod/fake-pod "pwd" "/tmp" || failed+=("${name}_target_exec")
    test_exec "${name}" default deploy/fake-pod "env" "TEST_ENV=test" || failed+=("${name}_cluster_default_exec")
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
