#!/usr/bin/env bash
# Copyright 2024 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

function update() {
  mapfile -t files < <(
    find . \
      -iname "*.input.yaml"
  )

  for file in "${files[@]}"; do
    update_stage "${file}"
  done
}

function update_stage() {
  local file="${1}"
  local stages=()
  local rel_path
  local base_dir
  local from
  base_dir="$(dirname "${file}")"

  from="$(grep '# @Stage: ' "${file}" | awk '{print $3}')"
  for line in ${from}; do
    rel_path="${line}"
    stages+=("${base_dir}/${rel_path}")
  done

  if [[ ${#stages[@]} -eq 0 ]]; then
    return
  fi

  go run ./hack/test_stage "${file}" "${stages[@]}" >"${file%.input.yaml}.output.yaml"
}

cd "${ROOT_DIR}" && update
