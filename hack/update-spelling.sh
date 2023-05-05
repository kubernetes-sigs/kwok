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

set -o errexit
set -o nounset
set -o pipefail

TOOL_VERSION="v0.3.4"

ROOT_DIR="$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)"
allowed_spelling_words="${ROOT_DIR}/hack/spelling.txt"

COMMAND=()
if command -v misspell; then
  COMMAND=(misspell)
elif command -v "${ROOT_DIR}/bin/misspell"; then
  COMMAND=("${ROOT_DIR}/bin/misspell")
else
  GOBIN="${ROOT_DIR}/bin/" go install github.com/client9/misspell/cmd/misspell@${TOOL_VERSION}
  COMMAND=("${ROOT_DIR}/bin/misspell")
fi

function update() {
  local ignore
  ignore="$(tr <"${allowed_spelling_words}" '\n' ',')"
  mapfile -t files < <(find . \( \
    -iname "*.md" -o \
    -iname "*.sh" -o \
    -iname "*.go" -o \
    -iname "*.tpl" -o \
    -iname "*.yaml" -o \
    -iname "*.yml" \
    \) \
    -not \( \
    -path ./.git/\* -o \
    -path ./vendor/\* -o \
    -path ./demo/node_modules/\* -o \
    -path ./site/themes/\* \
    \))
  "${COMMAND[@]}" -locale US -w -i "${ignore}" "${files[@]}"
}

cd "${ROOT_DIR}"
update || exit 1
