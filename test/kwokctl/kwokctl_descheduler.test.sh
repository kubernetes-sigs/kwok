#!/usr/bin/env bash
# Copyright 2026 The Kubernetes Authors.
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

DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")

source "${DIR}/helper.sh"

function main() {
  local all_releases=("${@}")
  for release in "${all_releases[@]}"; do
    name="test-kwokctl-descheduler-${release}"

    kwokctl create cluster --name "${name}" \
      --runtime=docker --enable descheduler \
      --extra-args=descheduler=descheduling-interval=10s

    kwokctl scale node --replicas 2 --name "${name}"

    api_ready=false
    for _ in {1..30}; do
      if kubectl get nodes >/dev/null 2>&1; then
        api_ready=true
        break
      fi
      sleep 2
    done
    if [[ "${api_ready}" != "true" ]]; then
      echo "Kubernetes API is not ready"
      exit 1
    fi

    sleep 5

    kubectl create -f "${DIR}/descheduler-case.yaml"

    kubectl wait --for=jsonpath='{.status.phase}'=Running pod/descheduler-trigger-a --timeout=60s
    kubectl wait --for=condition=Available deployment/descheduler-trigger-b --timeout=60s
    anchor_node="$(kubectl get pod descheduler-trigger-a -o jsonpath='{.spec.nodeName}')"
    violating_pod="$(kubectl get pods -l app=trigger-b -o jsonpath='{.items[0].metadata.name}')"
    violating_node="$(kubectl get pod "${violating_pod}" -o jsonpath='{.spec.nodeName}')"
    if [[ "${anchor_node}" != "${violating_node}" ]]; then
      echo "Trigger pods are not on the same node"
      exit 1
    fi

    kubectl label pod descheduler-trigger-a app=trigger-violation --overwrite

    descheduler_triggered=false
    for _ in {1..30}; do
      if ! kubectl get pod "${violating_pod}" >/dev/null 2>&1; then
        descheduler_triggered=true
        break
      fi
      sleep 2
    done

    if [[ "${descheduler_triggered}" != "true" ]]; then
      echo "Descheduler did not evict the violating pod"
      docker logs --tail 200 "kwok-${name}-descheduler" || true
      exit 1
    fi

    kwokctl delete cluster --name "${name}"
  done
}

requirements

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
