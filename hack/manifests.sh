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

ROOT_DIR="$(realpath "${DIR}/..")"

DEFAULT_IMAGE_PREFIX="registry.k8s.io/kwok"
BUCKET=""
GH_RELEASE=""
IMAGE_PREFIX="${DEFAULT_IMAGE_PREFIX}"
VERSION=""
DRY_RUN=false
PUSH=false
KUSTOMIZES=()

function usage() {
  echo "Usage: ${0} [--help] [--kustomize <kustomize> ...] [--bucket <bucket>] [--gh-release <gh-release>] [--image-prefix <image-prefix>] [--version <version>] [--push] [--dry-run]"
  echo "  --kustomize <kustomize> is directory of kustomize, is required"
  echo "  --bucket <bucket> is bucket to upload to"
  echo "  --gh-release <gh-release> is github release"
  echo "  --image-prefix <image-prefix> is kwok image prefix"
  echo "  --version <version> is version of binary"
  echo "  --push will push binary to bucket"
  echo "  --dry-run just show what would be done"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --kustomize | --kustomize=*)
      [[ "${arg#*=}" != "${arg}" ]] && KUSTOMIZES+=("${arg#*=}") || { KUSTOMIZES+=("${2}") && shift; } || :
      shift
      ;;
    --bucket | --bucket=*)
      [[ "${arg#*=}" != "${arg}" ]] && BUCKET="${arg#*=}" || { BUCKET="${2}" && shift; } || :
      shift
      ;;
    --gh-release | --gh-release=*)
      [[ "${arg#*=}" != "${arg}" ]] && GH_RELEASE="${arg#*=}" || { GH_RELEASE="${2}" && shift; } || :
      shift
      ;;
    --image-prefix | --image-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && IMAGE_PREFIX="${arg#*=}" || { IMAGE_PREFIX="${2}" && shift; } || :
      shift
      ;;
    --version | --version=*)
      [[ "${arg#*=}" != "${arg}" ]] && VERSION="${arg#*=}" || { VERSION="${2}" && shift; } || :
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

  if [[ "${#KUSTOMIZES}" -eq 0 ]]; then
    echo "--kustomize is required"
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
  dry_run cp -r "./kustomize" "./artifacts/"
  for kustomize in "${KUSTOMIZES[@]}"; do
    dry_run cd "./artifacts/kustomize/${kustomize}"
    dry_run kustomize edit set image "${DEFAULT_IMAGE_PREFIX}/kwok=${IMAGE_PREFIX}/kwok:${VERSION}"
    dry_run kustomize build "." -o "../../${kustomize}.yaml"
    dry_run cd -
    dry_run rm -r "./artifacts/kustomize"
    if [[ "${PUSH}" == "true" ]]; then
      if [[ "${BUCKET}" != "" ]]; then
        dry_run gsutil cp -P "./artifacts/${kustomize}.yaml" "${BUCKET}/releases/${VERSION}/manifests/${kustomize}.yaml"
      fi
      if [[ "${GH_RELEASE}" != "" ]]; then
        dry_run gh -R "${GH_RELEASE}" release upload "${VERSION}" "./artifacts/${kustomize}.yaml"
      fi
    fi
  done
}

args "$@"

cd "${ROOT_DIR}" && main
