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

ROOT_DIR=$(realpath $(dirname "${BASH_SOURCE[0]}")/..)

CODEGEN_PKG_VENDOR="${ROOT_DIR}"/vendor/k8s.io/code-generator
CODEGEN_PKG=${CODEGEN_PKG:-${CODEGEN_PKG_VENDOR}}
if [[ "${CODEGEN_PKG}" == "${CODEGEN_PKG_VENDOR}" ]]; then
  ls "${CODEGEN_PKG_VENDOR}" >/dev/null 2>&1 || go mod vendor
fi

KWOK_PROJECT="sigs.k8s.io/kwok"
KWOK_API_PACKAGE="${KWOK_PROJECT}/pkg/apis"

function codegen() {
  echo "Update codegen"

  bash "${CODEGEN_PKG}"/generate-internal-groups.sh \
    "deepcopy,defaulter,conversion" \
    "${KWOK_API_PACKAGE}" \
    "${KWOK_API_PACKAGE}" \
    "${KWOK_API_PACKAGE}" \
    ":v1alpha1,config/v1alpha1,internalversion" \
    --trim-path-prefix "${KWOK_PROJECT}" \
    --output-base "./" \
    --go-header-file "${ROOT_DIR}"/hack/boilerplate/boilerplate.go.txt
}

cd "${ROOT_DIR}" && codegen || exit 1
