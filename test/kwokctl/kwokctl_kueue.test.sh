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
    name="test-kwokctl-kueue-${release}"

    kwokctl create cluster --name "${name}" \
      --runtime=docker --enable kueue

    kwokctl scale node --replicas 1 --name "${name}"

    sleep 5
    kubectl create -f "${DIR}/kueue-case.yaml"

    sleep 5

    kubectl get workloads,job,pod

    if [[ $(kubectl get workloads) != *"kwok-cluster-queue"* ]]; then
      echo "Kueue workloads not found"
      exit 1
    fi

    if [[ $(kubectl get job) != *"Complete"* ]]; then
      echo "Kueue job not completed"
      exit 1
    fi

    if [[ $(kubectl get pod) != *"Completed"* ]]; then
      echo "Kueue pod not completed"
      exit 1
    fi

    kwokctl delete cluster --name "${name}"
  done
}

requirements

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
