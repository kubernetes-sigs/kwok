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

DIR=$(dirname "${BASH_SOURCE[0]}" | while IFS='' read -r line
do
    realpath "$line"
done)


source "${DIR}/helper.sh"

function main() {
  local all_releases=("${@}")
  for release in "${all_releases[@]}"; do
    KWOK_KUBE_VERSION="${release}" kwokctl -v=-4 create cluster --timeout 30m --wait 30m --quiet-pull --config "${DIR}"/kwokctl-config-runtimes.yaml || exit 1
    kwokctl delete cluster || exit 1
  done
}

requirements_for_binary

supported_releases | while IFS='' read -r line
do
    mian "$line"
done
