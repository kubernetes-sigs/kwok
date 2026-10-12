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
    name="test-kwokctl-jobset-${release}"

    kwokctl create cluster --name "${name}" \
      --runtime=docker --enable jobset

    kwokctl scale node --replicas 1 --name "${name}"

    if ! wait_for_webhook_admission "${DIR}/jobset-case.yaml" 60; then
      kwokctl logs jobset --name "${name}"
      exit 1
    fi
    kubectl create -f "${DIR}/jobset-case.yaml"

    if ! kubectl wait --for=create job/sample-jobset-workers-0 --timeout=60s; then
      echo "Error: JobSet child Job was not created within 60s" >&2
      kwokctl logs jobset --name "${name}" >&2 || true
      exit 1
    fi

    kubectl get jobsets,job,pod

    if [[ $(kubectl get jobsets) != *"sample-jobset"* ]]; then
      echo "JobSet not found"
      exit 1
    fi

    if [[ $(kubectl get job) != *"sample-jobset"* ]]; then
      echo "JobSet child job not found"
      exit 1
    fi

    kwokctl delete cluster --name "${name}"
  done
}

requirements

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
