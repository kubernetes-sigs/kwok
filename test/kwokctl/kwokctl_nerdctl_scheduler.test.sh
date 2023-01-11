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

ROOT_DIR="$(realpath "${DIR}/../..")"

source "${DIR}/helper.sh"
source "${ROOT_DIR}/hack/requirements.sh"

function requirements() {
  install_kubectl
  install_buildx
}

function main() {
  local all_releases=("${@}")
  build_kwokctl
  build_image_for_nerdctl

  test_all "nerdctl" "scheduler" "${all_releases[@]}" || exit 1
}

requirements

main $(supported_releases)
