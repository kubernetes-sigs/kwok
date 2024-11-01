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

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

DEFAULT_IMAGE_PREFIX="registry.k8s.io/kwok"
IMAGE="${DEFAULT_IMAGE_PREFIX}/charts"
DRY_RUN=false
PUSH=false
CHARTS=()

function usage() {
  echo "Usage: ${0} [--help] [--chart <chart> ...] [--bucket <bucket>] [--gh-release <gh-release>] [--image-prefix <image-prefix>] [--version <version>] [--push] [--dry-run]"
  echo "  --chart <chart> is directory of chart, is required"
  echo "  --image <image> is image, is required"
  echo "  --push will push binary to bucket"
  echo "  --dry-run just show what would be done"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --chart | --chart=*)
      [[ "${arg#*=}" != "${arg}" ]] && CHARTS+=("${arg#*=}") || { CHARTS+=("${2}") && shift; } || :
      shift
      ;;
    --image | --image=*)
      [[ "${arg#*=}" != "${arg}" ]] && IMAGE="${arg#*=}" || { IMAGE="${2}" && shift; } || :
      shift
      ;;
    --push | --push=*)
      [[ "${arg#*=}" != "${arg}" ]] && PUSH="${arg#*=}" || PUSH="true" || :
      shift
      ;;
    --dry-run | --dry-run=*)
      [[ "${arg#*=}" != "${arg}" ]] && DRY_RUN="${arg#*=}" || DRY_RUN="true" || :
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: ${arg}"
      usage
      exit 1
      ;;
    esac
  done

  if [[ "${#CHARTS}" -eq 0 ]]; then
    echo "--chart is required"
    usage
    exit 1
  fi
}

function dry_run() {
  echo "$*"
  if [[ "${DRY_RUN}" != "true" ]]; then
    eval "$*"
  fi
}

function main() {
  dry_run mkdir -p "./artifacts"
  for chart in "${CHARTS[@]}"; do
    dry_run helm package "./charts/${chart}" --destination "./artifacts"
    if [[ "${PUSH}" == "true" ]]; then
      dry_run helm push "./artifacts/${chart}-*.tgz" "oci://${IMAGE}"
    fi
  done
}

args "$@"

cd "${ROOT_DIR}" && main
