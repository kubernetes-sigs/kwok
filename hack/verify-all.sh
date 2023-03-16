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

failed=()

if [[ "${VERIFY_BOILERPLATE:-true}" == "true" ]]; then
  echo "[*] Verifying boilerplate..."
  "${ROOT_DIR}"/hack/verify-boilerplate.sh || failed+=(boilerplate)
fi

if [[ "${VERIFY_ENDS_NEWLINE:-true}" == "true" ]]; then
  echo "[*] Verifying ends newline..."
  "${ROOT_DIR}"/hack/verify-ends-newline.sh || failed+=(ends-newline)
fi

if [[ "${VERIFY_GO_MOD:-true}" == "true" ]]; then
  echo "[*] Verifying go mod..."
  "${ROOT_DIR}"/hack/verify-go-mod.sh || failed+=(go-mod)
fi

if [[ "${VERIFY_GO_FORMAT:-true}" == "true" ]]; then
  echo "[*] Verifying go format..."
  "${ROOT_DIR}"/hack/verify-go-format.sh || failed+=(go-format)
fi

if [[ "${VERIFY_CODEGEN:-true}" == "true" ]]; then
  echo "[*] Verifying codegen..."
  "${ROOT_DIR}"/hack/verify-codegen.sh || failed+=(codegen)
fi

if [[ "${VERIFY_CMD_DOCS:-true}" == "true" ]]; then
  echo "[*] Verifying cmd docs..."
  "${ROOT_DIR}"/hack/verify-cmd-docs.sh || failed+=(cmd-docs)
fi

# exit based on verify scripts
if [[ "${#failed[@]}" != 0 ]]; then
  echo "Verify failed for: ${failed[*]}"
  exit 1
fi
