#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
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

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/.."

function check_ends() {
  find . \
    -iname "*.md" \
    -o -iname "*.sh" \
    -o -iname "*.go" \
    -o -iname "*.tpl" \
    -o -iname "*.yaml" \
    -o -iname "*.yml" |
    xargs -I {} bash -c "[ -n \"\$(tail -c 1 {})\" ] && echo {}" || :
}

function check() {
  out="$(check_ends)"
  if [[ "${out}" != "" ]]; then
    echo "Add a new line in ends for below files"
    echo "${out}"
    return 1
  fi
}

cd "${ROOT_DIR}"

check || exit 1
