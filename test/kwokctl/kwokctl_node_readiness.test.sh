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
    name="test-kwokctl-node-readiness-controller-${release}"

    kwokctl create cluster --name "${name}" \
      --runtime=docker --enable node-readiness-controller

    kwokctl scale node --replicas 1 --name "${name}"

    sleep 5

    # Create NodeReadinessRule to trigger node-readiness-controller
    kubectl apply -f "${DIR}/node-readiness-case.yaml"

    sleep 10

    kubectl get nodereadinessrules

    # Check if NodeReadinessRule was created
    if ! kubectl get nodereadinessrules nodereadinessrule-sample &>/dev/null; then
      echo "NodeReadinessRule not found"
      exit 1
    fi

    # Check if node has the taint applied by node-readiness-controller
    if ! kubectl get node -o jsonpath='{.items[0].spec.taints[*].key}' | grep -q "readiness.k8s.io/NetworkReady"; then
      echo "NodeReadinessRule taint not applied to node"
      exit 1
    fi

    kwokctl delete cluster --name "${name}"
  done
}

requirements

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
