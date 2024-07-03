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

function sync_object_to_chart() {
  local src=$1
  local dest=$2

  sed \
    -e ':a;N;$!ba;s#metadata:\n  name: kwok-controller\n#metadata:\n  name: {{ include "kwok.fullname" . }}\n  labels:\n    {{- include "kwok.labels" . | nindent 4 }}\n#g' \
    -e 's#kwok-controller#{{ include "kwok.fullname" . }}#g' \
    -e 's#kube-system#{{ .Release.Namespace }}#g' \
    <"${src}" \
    >"${dest}"
}

function sync_stage_to_chart() {
  local src=$1
  local dest=$2

  sed \
    -e 's#{{#{{ `{{#g' \
    -e 's#}}#}}` }}#g' \
    -e "s#\`\"\"\`#\"\\\\\"\\\\\"\"#g" \
    <"${src}" \
    >"${dest}"
}

function sync_to_chart() {
  local src=$1
  local dest=$2

  cp "${src}" "${dest}"
}

function sync() {
  sync_object_to_chart kustomize/rbac/role.yaml charts/kwok/templates/role.yaml
  sync_object_to_chart kustomize/rbac/role_binding.yaml charts/kwok/templates/role_binding.yaml
  sync_object_to_chart kustomize/rbac/service_account.yaml charts/kwok/templates/service_account.yaml

  sync_stage_to_chart kustomize/stage/pod/fast/pod-ready.yaml charts/stage-fast/templates/pod-ready.yaml
  sync_stage_to_chart kustomize/stage/pod/fast/pod-complete.yaml charts/stage-fast/templates/pod-complete.yaml
  sync_stage_to_chart kustomize/stage/pod/fast/pod-delete.yaml charts/stage-fast/templates/pod-delete.yaml

  sync_stage_to_chart kustomize/stage/node/fast/node-initialize.yaml charts/stage-fast/templates/node-initialize.yaml
  sync_stage_to_chart kustomize/stage/node/heartbeat-with-lease/node-heartbeat-with-lease.yaml charts/stage-fast/templates/node-heartbeat-with-lease.yaml

  sync_stage_to_chart kustomize/metrics/resource/metrics-resource.yaml charts/metrics-usage/templates/metrics-resource.yaml
  sync_stage_to_chart kustomize/metrics/usage/usage-from-annotation.yaml charts/metrics-usage/templates/usage-from-annotation.yaml
}

cd "${ROOT_DIR}" && sync
