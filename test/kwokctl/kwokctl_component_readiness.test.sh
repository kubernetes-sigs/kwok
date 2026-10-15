#!/usr/bin/env bash
# Copyright 2026 The Kubernetes Authors.
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

DIR="$(realpath "${DIR}")"

source "${DIR}/helper.sh"

# Resources owned by the current iteration, released by cleanup.
CLUSTER_NAME=""
JOBSET_CONTAINER=""
UNPAUSE_PID=""

function stop_scheduled_unpause() {
  if [[ -z "${UNPAUSE_PID}" ]]; then
    return 0
  fi
  # Stop the sleep first, so the subshell does not go on to unpause.
  pkill -TERM -P "${UNPAUSE_PID}" 2>/dev/null || :
  kill -TERM "${UNPAUSE_PID}" 2>/dev/null || :
  wait "${UNPAUSE_PID}" 2>/dev/null || :
  UNPAUSE_PID=""
}

# cleanup is idempotent: it only touches the cluster created by this script.
function cleanup() {
  local ret=0
  stop_scheduled_unpause
  if [[ -z "${CLUSTER_NAME}" ]]; then
    return 0
  fi
  if [[ "$(docker inspect --format '{{.State.Paused}}' "${JOBSET_CONTAINER}" 2>/dev/null)" == "true" ]]; then
    docker unpause "${JOBSET_CONTAINER}" >/dev/null || ret=1
  fi
  kwokctl delete cluster --name "${CLUSTER_NAME}" || ret=1
  CLUSTER_NAME=""
  JOBSET_CONTAINER=""
  return "${ret}"
}

function on_exit() {
  local status="${?}"
  if ! cleanup; then
    echo "Error: failed to clean up the test cluster" >&2
    if [[ "${status}" -eq 0 ]]; then
      status=1
    fi
  fi
  exit "${status}"
}

trap on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

function component_status() {
  local name="${1}"
  local component="${2}"
  kwokctl get components --name "${name}" --output wide | awk -v c="${component}" '$1 == c { print $2 }'
}

function wait_component_status() {
  local name="${1}"
  local component="${2}"
  local want="${3}"
  local got
  local i
  for ((i = 0; i < 30; i++)); do
    got="$(component_status "${name}" "${component}")"
    if [[ "${got}" == "${want}" ]]; then
      return 0
    fi
    sleep 1
  done
  echo "Component ${component} status is ${got}, want ${want}"
  kwokctl get components --name "${name}" --output wide
  return 1
}

function main() {
  local all_releases=("${@}")
  for release in "${all_releases[@]}"; do
    name="test-kwokctl-component-readiness-${release}"

    # Never take ownership of a cluster this run did not create.
    if kwokctl get clusters | grep -qxF "${name}"; then
      echo "Cluster ${name} already exists"
      exit 1
    fi
    CLUSTER_NAME="${name}"
    JOBSET_CONTAINER="kwok-${name}-jobset"

    if ! kwokctl create cluster --name "${name}" \
      --runtime=docker --enable jobset --wait 2m; then
      echo "Failed to create cluster"
      exit 1
    fi

    # --wait returned, so the JobSet webhook must already be serving.
    if [[ "$(component_status "${name}" jobset)" != "Ready" ]]; then
      echo "JobSet is not ready after --wait"
      kwokctl get components --name "${name}" --output wide
      exit 1
    fi
    if ! kubectl create --dry-run=server -f "${DIR}/jobset-case.yaml"; then
      echo "JobSet webhook is not serving after --wait"
      exit 1
    fi

    # A paused container is still running but does not answer the probe.
    docker pause "${JOBSET_CONTAINER}" || exit 1
    wait_component_status "${name}" jobset NotReady || exit 1

    # --wait must block until the probe passes again.
    (
      sleep 10 && docker unpause "${JOBSET_CONTAINER}"
    ) &
    UNPAUSE_PID="${!}"
    local start="${SECONDS}"
    if ! kwokctl start cluster --name "${name}" --wait 2m; then
      echo "Failed to start cluster"
      exit 1
    fi
    local elapsed="$((SECONDS - start))"
    if ! wait "${UNPAUSE_PID}"; then
      UNPAUSE_PID=""
      echo "Failed to unpause JobSet"
      exit 1
    fi
    UNPAUSE_PID=""
    if [[ "${elapsed}" -lt 8 ]]; then
      echo "--wait returned after ${elapsed}s, before JobSet was unpaused"
      exit 1
    fi
    if [[ "$(component_status "${name}" jobset)" != "Ready" ]]; then
      echo "JobSet is not ready after unpause"
      kwokctl get components --name "${name}" --output wide
      exit 1
    fi
    if ! kubectl create --dry-run=server -f "${DIR}/jobset-case.yaml"; then
      echo "JobSet webhook is not serving after unpause"
      exit 1
    fi

    docker stop "${JOBSET_CONTAINER}" || exit 1
    wait_component_status "${name}" jobset Stopped || exit 1

    # Components without a probe keep the previous behavior.
    if [[ "$(component_status "${name}" kube-scheduler)" != "Ready" ]]; then
      echo "kube-scheduler should be Ready"
      exit 1
    fi

    if ! cleanup; then
      echo "Failed to delete cluster ${name}"
      exit 1
    fi
  done
}

requirements

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
