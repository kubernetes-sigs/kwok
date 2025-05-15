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

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

function controller-gen() {
  go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.1 "$@"
}

function gen() {
  rm -rf \
    "${ROOT_DIR}/kustomize/crd/bases" \
    "${ROOT_DIR}/kustomize/rbac/rbac.yaml"
  echo "Generating crd/rbac"
  controller-gen \
    rbac:roleName=kwok-controller \
    crd:allowDangerousTypes=true \
    paths=./pkg/apis/v1alpha1/ \
    output:crd:artifacts:config=kustomize/crd/bases \
    output:rbac:artifacts:config=kustomize/rbac
}

cd "${ROOT_DIR}" && gen
