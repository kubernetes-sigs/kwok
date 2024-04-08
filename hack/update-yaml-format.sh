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

function format() {
  echo "Update yaml format"
  mapfile -t findfiles < <(find . \( \
    -iname "*.yaml" \
    -o -iname "*.yml" \
    \) \
    -not \( \
    -path ./vendor/\* \
    -o -path ./demo/node_modules/\* \
    -o -path ./site/themes/\* \
    -o -path ./kustomize/crd/bases/\* \
    -o -path ./kustomize/rbac/\* \
    \))
  go run github.com/google/yamlfmt/cmd/yamlfmt@v0.11.0 -conf .yamlfmt.yaml "${findfiles[@]}"
}

cd "${ROOT_DIR}" && format
