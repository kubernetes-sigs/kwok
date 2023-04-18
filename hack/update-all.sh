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

ROOT_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)

failed=()

if [[ "${UPDATE_CODEGEN:-true}" == "true" ]]; then
  echo "[*] Update codegen..."
  "${ROOT_DIR}"/hack/update-codegen.sh || failed+=(codegen)
fi

if [[ "${UPDATE_GO_FORMAT:-true}" == "true" ]]; then
  echo "[*] Update go format..."
  "${ROOT_DIR}"/hack/update-go-format.sh || failed+=(go-format)
fi

if [[ "${UPDATE_GO_MOD:-true}" == "true" ]]; then
  echo "[*] Update go mod..."
  "${ROOT_DIR}"/hack/update-go-mod.sh || failed+=(go-mod)
fi

if [[ "${UPDATE_ENDS_NEWLINE:-true}" == "true" ]]; then
  echo "[*] Update ends newline..."
  "${ROOT_DIR}"/hack/update-ends-newline.sh || failed+=(ends-newline)
fi

if [[ "${UPDATE_CMD_DOCS:-true}" == "true" ]]; then
  echo "[*] Update cmd docs..."
  "${ROOT_DIR}"/hack/update-cmd-docs.sh || failed+=(cmd-docs)
fi

if [[ "${UPDATE_SHELL_FORMAT:-true}" == "true" ]]; then
  echo "[*] Update shell format..."
  "${ROOT_DIR}"/hack/update-shell-format.sh || failed+=(shell-format)
fi

if [[ "${#failed[@]}" != 0 ]]; then
  echo "Update failed for: ${failed[*]}"
  exit 1
fi
