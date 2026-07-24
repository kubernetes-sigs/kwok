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
    name="test-kwokctl-podgroup-${release}"

    kwokctl create cluster --name "${name}" \
      --enable scheduler-plugins \
      --kube-scheduler-config "${DIR}/podgroup-scheduler-config.yaml"

    kwokctl scale node --replicas 2 --name "${name}"

    sleep 2
    kubectl create -f "${DIR}/podgroup-case.yaml"

    sleep 10

    kubectl get pg,pod

    if [[ $(kubectl get pg) != *"Running"* ]]; then
      echo "podgroup not running"
      exit 1
    fi

    if [[ $(kubectl get pod) != *"Running"* ]]; then
      echo "pod not running"
      exit 1
    fi

    kwokctl delete cluster --name "${name}"
  done
}

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
