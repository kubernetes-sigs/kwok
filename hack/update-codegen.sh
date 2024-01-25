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

function deepcopy-gen() {
  go run k8s.io/code-generator/cmd/deepcopy-gen "$@"
}

function defaulter-gen() {
  go run k8s.io/code-generator/cmd/defaulter-gen "$@"
}

function conversion-gen() {
  go run k8s.io/code-generator/cmd/conversion-gen "$@"
}

function client-gen() {
  go run k8s.io/code-generator/cmd/client-gen "$@"
}

function gen() {
  rm -rf \
    "${ROOT_DIR}/pkg/apis/internalversion"/zz_generated.*.go \
    "${ROOT_DIR}/pkg/apis/v1alpha1"/zz_generated.*.go \
    "${ROOT_DIR}/pkg/apis/config/v1alpha1"/zz_generated.*.go \
    "${ROOT_DIR}/pkg/apis/action/v1alpha1"/zz_generated.*.go
  echo "Generating deepcopy"
  deepcopy-gen \
    --input-dirs ./pkg/apis/internalversion/ \
    --input-dirs ./pkg/apis/v1alpha1/ \
    --input-dirs ./pkg/apis/config/v1alpha1/ \
    --input-dirs ./pkg/apis/action/v1alpha1/ \
    --trim-path-prefix sigs.k8s.io/kwok/pkg/apis \
    --output-file-base zz_generated.deepcopy \
    --go-header-file ./hack/boilerplate/boilerplate.generatego.txt
  echo "Generating defaulter"
  defaulter-gen \
    --input-dirs ./pkg/apis/v1alpha1/ \
    --input-dirs ./pkg/apis/config/v1alpha1/ \
    --input-dirs ./pkg/apis/action/v1alpha1/ \
    --trim-path-prefix sigs.k8s.io/kwok/pkg/apis \
    --output-file-base zz_generated.defaults \
    --go-header-file ./hack/boilerplate/boilerplate.generatego.txt
  echo "Generating conversion"
  conversion-gen \
    --input-dirs ./pkg/apis/internalversion/ \
    --trim-path-prefix sigs.k8s.io/kwok/pkg/apis \
    --output-file-base zz_generated.conversion \
    --go-header-file ./hack/boilerplate/boilerplate.generatego.txt

  rm -rf "${ROOT_DIR}/pkg/client"
  echo "Generating client"
  client-gen \
    --clientset-name versioned \
    --input-base "" \
    --input sigs.k8s.io/kwok/pkg/apis/v1alpha1 \
    --output-package sigs.k8s.io/kwok/pkg/client/clientset \
    --go-header-file ./hack/boilerplate/boilerplate.generatego.txt \
    --plural-exceptions="Logs:Logs,ClusterLogs:ClusterLogs"
}

cd "${ROOT_DIR}" && gen
