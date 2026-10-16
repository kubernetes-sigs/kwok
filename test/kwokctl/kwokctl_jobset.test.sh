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

    # --wait blocks until JobSet's /readyz passes, which requires its webhook server to be serving.
    kwokctl create cluster --name "${name}" \
      --runtime=docker --enable jobset --wait=2m

    # --wait logs a timeout but does not fail, so check JobSet explicitly.
    if [[ "$(kwokctl get components --name "${name}" --output wide | awk '$1 == "jobset" { print $2 }')" != "Ready" ]]; then
      echo "JobSet is not ready"
      kwokctl get components --name "${name}" --output wide
      kwokctl logs jobset --name "${name}"
      exit 1
    fi

    kwokctl scale node --replicas 1 --name "${name}"

    kubectl create -f "${DIR}/jobset-case.yaml"

    if ! kubectl wait --for=create job/sample-jobset-workers-0 --timeout=60s; then
      echo "JobSet child job was not created"
      kubectl get jobsets,job,pod
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
