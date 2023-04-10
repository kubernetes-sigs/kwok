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

ARGS=("$@")

TMPDIR="$(mktemp -d)"

ORIGIN_SNAPSHOT="${TMPDIR}/snapshot-original.yaml"
MIRROR_OLD_SNAPSHOT="${TMPDIR}/snapshot-mirror-old.yaml"
MIRROR_NEW_SNAPSHOT="${TMPDIR}/snapshot-mirror-new.yaml"
NAME="dryrun-${RANDOM}"
KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"
SNAPSHOT_CACHE=""
DURATION=5

function usage() {
  echo "Usage: $0 [options] [kubectl args]"
  echo "Options:"
  echo "  --kubeconfig=<path>  Path to kubeconfig file"
  echo "  --snapshot-cache=<path> Path to snapshot cache file"
  echo "  --duration=<seconds> Duration to wait for the mirror cluster to be ready"
  echo "  --help               Print this help message"
}

function args() {
  local arg
  local passed_args=()
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --kubeconfig | --kubeconfig=* )
      [[ "${arg#*=}" != "${arg}" ]] && KUBECONFIG="${kubeconfig#*=}" || { KUBECONFIG="${2}" && shift; }
      shift
      ;;
    --duration | --duration=* )
      [[ "${arg#*=}" != "${arg}" ]] && DURATION="${arg#*=}" || { DURATION="${2}" && shift; }
      shift
      ;;
    --snapshot-cache | --snapshot-cache=* )
      [[ "${arg#*=}" != "${arg}" ]] && SNAPSHOT_CACHE="${arg#*=}" || { SNAPSHOT_CACHE="${2}" && shift; }
      shift
      ;;
    --help )
      usage
      exit 0
      ;;
    *)
      passed_args+=("${arg}")
      shift
      ;;
    esac
  done
  ARGS=("${passed_args[@]}")
}

function log() {
  echo "------ $* -------" >&2
}

function cleanup() {
  log Clean up
  kwokctl --name "${NAME}" delete cluster >&2
  rm -rf "${TMPDIR}" >&2
}

function filter() {
  grep -v ' resourceVersion: '
}

function main() {
  if [[ "${SNAPSHOT_CACHE}" != "" ]]; then
    if [[ ! -f "${SNAPSHOT_CACHE}" ]]; then
      log "Exporting Snapshot from Cluster to ${SNAPSHOT_CACHE}"
      kwokctl snapshot export --kubeconfig "${KUBECONFIG}" --path "${SNAPSHOT_CACHE}" >&2
    else
      log "Using Cached Snapshot from ${SNAPSHOT_CACHE}"
    fi
    ORIGIN_SNAPSHOT="${SNAPSHOT_CACHE}"
  else
    log "Exporting Snapshot from Cluster"
    kwokctl snapshot export --kubeconfig "${KUBECONFIG}" --path "${ORIGIN_SNAPSHOT}" >&2
  fi

  trap cleanup EXIT
  log "Creating Mirror Cluster ${KWOK_KUBE_VERSION:-}"
  kwokctl --name ${NAME} create cluster --kubeconfig '' >&2

  log "Restoring Snapshot to Mirror Cluster"
  kwokctl --name ${NAME} snapshot restore --format k8s --path "${ORIGIN_SNAPSHOT}" >&2

  log "Waiting for Mirror Cluster to react fully"
  sleep "${DURATION}"

  log "Save Old Snapshot from Mirror Cluster"
  kwokctl --name "${NAME}" snapshot save --format k8s --path "${MIRROR_OLD_SNAPSHOT}" >&2

  log "Dry Run kubectl ${ARGS[*]}"
  kwokctl --name "${NAME}" kubectl "${ARGS[@]}" >&2

  log "Waiting for Dry Run to react fully"
  sleep "${DURATION}"

  log "Save New Snapshot from Mirror Cluster"
  kwokctl --name "${NAME}" snapshot save --format k8s --path "${MIRROR_NEW_SNAPSHOT}" >&2

  cat "${MIRROR_OLD_SNAPSHOT}" | filter > "${MIRROR_OLD_SNAPSHOT}.clean"
  cat "${MIRROR_NEW_SNAPSHOT}" | filter > "${MIRROR_NEW_SNAPSHOT}.clean"

  log "Diff between Old and New Snapshot"
  echo ================================================ >&2
  diff -u "${MIRROR_OLD_SNAPSHOT}.clean" "${MIRROR_NEW_SNAPSHOT}.clean"
  echo ================================================ >&2
  log "Done"
}

args "$@"
main
