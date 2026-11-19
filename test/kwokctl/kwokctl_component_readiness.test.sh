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

KWOK_RUNTIME="${KWOK_RUNTIME:-docker}"

# Resources owned by the current iteration, released by cleanup.
CLUSTER_NAME=""
# FROZEN holds "<component> <pid>" entries, the pid is empty for docker.
FROZEN=()
RESUME_PID=""
START_PID=""

function kind_node() {
  echo "kwok-${CLUSTER_NAME}-control-plane"
}

# process_name returns the process name of a component in the kind node.
function process_name() {
  local component="${1}"
  case "${component}" in
  kwok-controller)
    echo "kwok"
    ;;
  *)
    echo "Unsupported component ${component} for kind" >&2
    return 1
    ;;
  esac
}

# kind_component_pid prints the pid, in the kind node, of the only process of the component.
# The node image may not ship procps, so the process is found by its name in /proc.
# It refuses to match anything else, such as the kubelet or the init of the node.
function kind_component_pid() {
  local component="${1}"
  local comm
  comm="$(process_name "${component}")" || return 1
  docker exec "$(kind_node)" sh -c '
    pids=""
    for p in /proc/[0-9]*; do
      if [ "$(cat "${p}/comm" 2>/dev/null)" = "'"${comm}"'" ]; then
        pids="${pids} ${p#/proc/}"
      fi
    done
    set -- ${pids}
    if [ "${#}" -ne 1 ] || [ "${1}" = "1" ]; then
      echo "Want exactly one '"${comm}"' process, found:${pids}" >&2
      exit 1
    fi
    echo "${1}"
  '
}

# signal_kind_pid signals a pid in the kind node, only while it is still the process of the component.
function signal_kind_pid() {
  local component="${1}"
  local pid="${2}"
  local signal="${3}"
  local comm
  comm="$(process_name "${component}")" || return 1
  docker exec "$(kind_node)" sh -c '
    if [ "$(cat /proc/'"${pid}"'/comm 2>/dev/null)" = "'"${comm}"'" ]; then
      kill -'"${signal}"' '"${pid}"'
    fi
  '
}

# freeze stops a component from answering while it keeps running.
function freeze() {
  local component="${1}"
  local pid=""
  # Recorded before the component is frozen, so that cleanup always resumes it.
  case "${KWOK_RUNTIME}" in
  docker)
    FROZEN+=("${component} ")
    docker pause "kwok-${CLUSTER_NAME}-${component}" >/dev/null || return 1
    ;;
  binary)
    pid="$(cat "${KWOK_WORKDIR}/clusters/${CLUSTER_NAME}/pids/${component}.pid")" || return 1
    FROZEN+=("${component} ${pid}")
    kill -STOP "${pid}" || return 1
    ;;
  kind)
    pid="$(kind_component_pid "${component}")" || return 1
    FROZEN+=("${component} ${pid}")
    signal_kind_pid "${component}" "${pid}" STOP || return 1
    ;;
  esac
}

# resume continues a component frozen by freeze.
function resume() {
  local component="${1}"
  local entry
  local pid
  for entry in "${FROZEN[@]}"; do
    if [[ "${entry%% *}" != "${component}" ]]; then
      continue
    fi
    pid="${entry#* }"
    case "${KWOK_RUNTIME}" in
    docker)
      if [[ "$(docker inspect --format '{{.State.Paused}}' "kwok-${CLUSTER_NAME}-${component}" 2>/dev/null)" == "true" ]]; then
        docker unpause "kwok-${CLUSTER_NAME}-${component}" >/dev/null || return 1
      fi
      ;;
    binary)
      kill -CONT "${pid}" 2>/dev/null || :
      ;;
    kind)
      signal_kind_pid "${component}" "${pid}" CONT || return 1
      ;;
    esac
  done
}

function stop_scheduled_resume() {
  if [[ -z "${RESUME_PID}" ]]; then
    return 0
  fi
  # Stop the sleep first, so the subshell does not go on to resume.
  pkill -TERM -P "${RESUME_PID}" 2>/dev/null || :
  kill -TERM "${RESUME_PID}" 2>/dev/null || :
  wait "${RESUME_PID}" 2>/dev/null || :
  RESUME_PID=""
}

# cleanup is idempotent: it only touches the cluster created by this script.
function cleanup() {
  local ret=0
  local entry
  stop_scheduled_resume
  if [[ -n "${START_PID}" ]]; then
    kill -TERM "${START_PID}" 2>/dev/null || :
    wait "${START_PID}" 2>/dev/null || :
    START_PID=""
  fi
  if [[ -z "${CLUSTER_NAME}" ]]; then
    return 0
  fi
  # A stopped process does not handle the termination signal until it is continued.
  for entry in "${FROZEN[@]}"; do
    resume "${entry%% *}" || ret=1
  done
  FROZEN=()
  kwokctl delete cluster --name "${CLUSTER_NAME}" || ret=1
  CLUSTER_NAME=""
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
  for ((i = 0; i < 60; i++)); do
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

# assert_all_ready checks the status independently, since --wait does not fail on a timeout.
function assert_all_ready() {
  local name="${1}"
  local reason="${2}"
  local not_ready
  not_ready="$(kwokctl get components --name "${name}" --output wide | awk 'NR > 1 && $2 != "Ready" { print $1 "=" $2 }')"
  if [[ -n "${not_ready}" ]]; then
    echo "Components are not ready ${reason}: ${not_ready}"
    kwokctl get components --name "${name}" --output wide
    return 1
  fi
}

# assert_frozen checks that only the target is frozen, and that it is still running while not ready.
function assert_frozen() {
  local name="${1}"
  local component="${2}"
  case "${KWOK_RUNTIME}" in
  binary)
    local pid
    pid="$(cat "${KWOK_WORKDIR}/clusters/${name}/pids/${component}.pid")" || return 1
    if [[ "$(ps -o stat= -p "${pid}")" != T* ]]; then
      echo "${component} process ${pid} is not stopped"
      return 1
    fi
    ;;
  kind)
    # The kubelet and the control plane must keep working, which also lets the pod status be read.
    local node_ready
    node_ready="$(kubectl get node "$(kind_node)" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')"
    if [[ "${node_ready}" != "True" ]]; then
      echo "Node $(kind_node) is not ready while only ${component} is frozen"
      return 1
    fi
    local pod_status
    pod_status="$(kubectl get pod -n kube-system "${component}-$(kind_node)" -o jsonpath='{.status.phase}/{.status.containerStatuses[0].ready}/{.status.containerStatuses[0].state.running.startedAt}')"
    if [[ "${pod_status}" != Running/false/?* ]]; then
      echo "Pod ${component} should be running and not ready, got ${pod_status}"
      kubectl get pod -n kube-system "${component}-$(kind_node)" -o yaml
      return 1
    fi
    ;;
  esac
}

# test_probe_follows_freeze checks that a probed component is not ready while frozen, and ready once resumed.
function test_probe_follows_freeze() {
  local name="${1}"
  local component="${2}"
  freeze "${component}" || return 1
  wait_component_status "${name}" "${component}" NotReady || return 1
  resume "${component}" || return 1
  wait_component_status "${name}" "${component}" Ready || return 1
}

# test_wait_blocks_until_resumed checks that --wait blocks while the component does not answer.
function test_wait_blocks_until_resumed() {
  local name="${1}"
  local component="${2}"

  freeze "${component}" || return 1
  wait_component_status "${name}" "${component}" NotReady || return 1
  assert_frozen "${name}" "${component}" || return 1

  (
    sleep 10 && resume "${component}"
  ) &
  RESUME_PID="${!}"
  local start="${SECONDS}"
  # Wait in the background, so that a signal runs the cleanup without waiting for kwokctl.
  kwokctl start cluster --name "${name}" --wait 2m &
  START_PID="${!}"
  if ! wait "${START_PID}"; then
    START_PID=""
    echo "Failed to start cluster"
    return 1
  fi
  START_PID=""
  local elapsed="$((SECONDS - start))"
  if ! wait "${RESUME_PID}"; then
    RESUME_PID=""
    echo "Failed to resume ${component}"
    return 1
  fi
  RESUME_PID=""
  if [[ "${elapsed}" -lt 8 ]]; then
    echo "--wait returned after ${elapsed}s, before ${component} was resumed"
    return 1
  fi
  assert_all_ready "${name}" "after ${component} was resumed" || return 1
}

function test_docker() {
  local name="${1}"

  # --wait returned, so the JobSet webhook must already be serving.
  if ! kubectl create --dry-run=server -f "${DIR}/jobset-case.yaml"; then
    echo "JobSet webhook is not serving after --wait"
    return 1
  fi

  test_wait_blocks_until_resumed "${name}" jobset || return 1
  if ! kubectl create --dry-run=server -f "${DIR}/jobset-case.yaml"; then
    echo "JobSet webhook is not serving after unpause"
    return 1
  fi

  docker stop "kwok-${name}-jobset" || return 1
  wait_component_status "${name}" jobset Stopped || return 1

  test_probe_follows_freeze "${name}" kube-scheduler || return 1
  test_probe_follows_freeze "${name}" kube-controller-manager || return 1

  # kube-apiserver is probed over HTTPS through the nc relay.
  # This runs last, since components may lose their leader lease while kube-apiserver is frozen.
  freeze kube-apiserver || return 1
  wait_component_status "${name}" kube-apiserver NotReady || return 1
  resume kube-apiserver || return 1
  wait_component_status "${name}" kube-apiserver Ready || return 1
}

function test_binary() {
  local name="${1}"

  test_wait_blocks_until_resumed "${name}" kwok-controller || return 1

  kill -KILL "$(cat "${KWOK_WORKDIR}/clusters/${name}/pids/kwok-controller.pid")" || return 1
  wait_component_status "${name}" kwok-controller Stopped || return 1

  test_probe_follows_freeze "${name}" kube-scheduler || return 1
  test_probe_follows_freeze "${name}" kube-controller-manager || return 1

  # The proxy process keeps running, but cannot serve while kube-apiserver does not answer.
  # This runs last, since components may lose their leader lease while kube-apiserver is frozen.
  freeze kube-apiserver || return 1
  wait_component_status "${name}" kube-apiserver NotReady || return 1
  wait_component_status "${name}" kube-apiserver-insecure-proxy NotReady || return 1
  resume kube-apiserver || return 1
  wait_component_status "${name}" kube-apiserver Ready || return 1
  wait_component_status "${name}" kube-apiserver-insecure-proxy Ready || return 1
}

function test_kind() {
  local name="${1}"

  test_wait_blocks_until_resumed "${name}" kwok-controller || return 1
}

function main() {
  local all_releases=("${@}")
  local args=(
    --runtime="${KWOK_RUNTIME}"
    --prometheus-port=19090
    --kube-apiserver-insecure-port=18001
  )
  case "${KWOK_RUNTIME}" in
  docker)
    args+=(--enable=jobset)
    ;;
  binary | kind) ;;
  *)
    echo "Unsupported runtime ${KWOK_RUNTIME}"
    exit 1
    ;;
  esac

  for release in "${all_releases[@]}"; do
    # kind derives the node hostname from the name, which must stay a valid DNS label.
    name="kwokctl-readiness-${KWOK_RUNTIME}-${release//./-}"

    # Never take ownership of a cluster this run did not create.
    if kwokctl get clusters | grep -qxF "${name}"; then
      echo "Cluster ${name} already exists"
      exit 1
    fi
    CLUSTER_NAME="${name}"

    if ! KWOK_KUBE_VERSION="${release}" kwokctl create cluster --name "${name}" "${args[@]}" --wait 5m; then
      echo "Failed to create cluster"
      exit 1
    fi

    assert_all_ready "${name}" "after --wait" || exit 1

    # kube-scheduler is probed on binary and docker, and runs under kubeadm on kind.
    if [[ "$(component_status "${name}" kube-scheduler)" != "Ready" ]]; then
      echo "kube-scheduler should be Ready"
      exit 1
    fi

    "test_${KWOK_RUNTIME}" "${name}" || exit 1

    if ! cleanup; then
      echo "Failed to delete cluster ${name}"
      exit 1
    fi
  done
}

case "${KWOK_RUNTIME}" in
binary)
  requirements_for_binary
  ;;
*)
  requirements
  ;;
esac

mapfile -t releases < <(supported_releases)
main "${releases[@]}"
