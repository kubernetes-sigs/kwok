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

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/../..")"

export KWOK_LOGS_DIR="${ROOT_DIR}/logs"

function save_logs() {
  local name="${1}"
  mkdir -p "${KWOK_LOGS_DIR}"
  kwokctl --name="${name}" export logs "${KWOK_LOGS_DIR}"
}

function create_cluster() {
  local name="${1}"
  local release="${2}"
  shift 2

  if ! KWOK_KUBE_VERSION="${release}" kwokctl \
    create cluster \
    --name "${name}" \
    --timeout 30m \
    --wait 1m \
    --quiet-pull \
    --disable-qps-limits \
    --network=host \
    "$@"; then
    echo "Error: Cluster ${name} creation failed"
    exit 1
  fi
}

function delete_cluster() {
  local name="${1}"
  save_logs "${name}"
  if ! kwokctl delete cluster --name "${name}"; then
    echo "Error: Cluster ${name} deletion failed"
    exit 1
  fi
}

function child_timeout() {
  local to="${1}"
  shift
  "${@}" &
  local wp=$!
  local start=0
  while kill -0 "${wp}" 2>/dev/null; do
    if [[ "${start}" -ge "${to}" ]]; then
      kill "${wp}"
      echo "Error: Timeout ${to}s" >&2
      return 1
    fi
    ((start++))
    sleep 1
  done
  echo "Took ${start}s" >&2
}

function retry() {
  local times="${1}"
  shift
  local start=0
  while true; do
    if "${@}"; then
      return 0
    fi
    if [[ "${start}" -ge "${times}" ]]; then
      echo "Error: Retry ${times} times" >&2
      return 1
    fi
    ((start++))
    sleep 1
  done
}
