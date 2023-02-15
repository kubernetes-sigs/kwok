#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
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
DIR="$(realpath "${DIR}")"
ROOT_DIR="$(realpath "${DIR}/../..")"
DOCKERFILE="$(realpath "${DIR}/Dockerfile" --relative-to="${ROOT_DIR}")"

DRY_RUN=false
PUSH=false
IMAGES=()
EXTRA_TAGS=()
PLATFORMS=()
VERSION=""
STAGING_PREFIX=""
BUILDER="docker"

function usage() {
  echo "Usage: ${0} [--help] [--version <version>] [--image <image> ...] [--extra-tag <extra-tag> ...] [--staging-prefix <staging-prefix>] [--platform <platform> ...] [--push] [--dry-run] [--builder <builder>]"
  echo "  --version <version> is kwok version, is required"
  echo "  --image <image> is image, is required"
  echo "  --extra-tag <extra-tag> is extra tag"
  echo "  --staging-prefix <staging-prefix> is staging prefix for tag"
  echo "  --platform <platform> is multi-platform capable for image"
  echo "  --push will push image to registry"
  echo "  --dry-run just show what would be done"
  echo "  --builder <builder> specify image builder, default: docker. available options: docker, nerdctl"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --version | --version=*)
      [[ "${arg#*=}" != "${arg}" ]] && VERSION="${arg#*=}" || { VERSION="${2}" && shift; }
      shift
      ;;
    --image | --image=*)
      [[ "${arg#*=}" != "${arg}" ]] && IMAGES+=("${arg#*=}") || { IMAGES+=("${2}") && shift; }
      shift
      ;;
    --extra-tag | --extra-tag=*)
      [[ "${arg#*=}" != "${arg}" ]] && EXTRA_TAGS+=("${arg#*=}") || { EXTRA_TAGS+=("${2}") && shift; }
      shift
      ;;
    --staging-prefix | --staging-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && STAGING_PREFIX="${arg#*=}" || { STAGING_PREFIX="${2}" && shift; }
      shift
      ;;
    --platform | --platform=*)
      [[ "${arg#*=}" != "${arg}" ]] && PLATFORMS+=("${arg#*=}") || { PLATFORMS+=("${2}") && shift; }
      shift
      ;;
    --push | --push=*)
      [[ "${arg#*=}" != "${arg}" ]] && PUSH="${arg#*=}" || PUSH="true"
      shift
      ;;
    --dry-run | --dry-run=*)
      [[ "${arg#*=}" != "${arg}" ]] && DRY_RUN="${arg#*=}" || DRY_RUN="true"
      shift
      ;;
    --builder | --builder=*)
      [[ "${arg#*=}" != "${arg}" ]] && BUILDER="${arg#*=}" || { BUILDER="${2}" && shift; }
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

  if [[ "${VERSION}" == "" ]]; then
    echo "--version is required"
    usage
    exit 1
  fi

  if [[ "${#IMAGES}" -eq 0 ]]; then
    echo "--image is required"
    usage
    exit 1
  fi

  if [[ "${#PLATFORMS[*]}" -eq 0 ]]; then
    PLATFORMS+=(
      linux/amd64
    )
  fi
}

function dry_run() {
  echo "${@}"
  if [[ "${DRY_RUN}" != "true" ]]; then
    eval "${@}"
  fi
}

function main() {
  local extra_args
  local platform_args
  local images
  local tag

  extra_args=()
  images=()
  for image in "${IMAGES[@]}"; do
    tag="${VERSION}"
    if [[ "${STAGING_PREFIX}" != "" ]]; then
      tag="${STAGING_PREFIX}-${VERSION}"
    fi
    extra_args+=("--tag=${image}:${tag}")
    images+=("${image}:${tag}")

    if [[ "${#EXTRA_TAGS[@]}" -ne 0 ]]; then
      for extra_tag in "${EXTRA_TAGS[@]}"; do
        tag="${extra_tag}"
        if [[ "${STAGING_PREFIX}" != "" ]]; then
          tag="${STAGING_PREFIX}-${extra_tag}"
        fi
        extra_args+=("--tag=${image}:${tag}")
        images+=("${image}:${tag}")
      done
    fi
  done

  for platform in "${PLATFORMS[@]}"; do
    extra_args+=("--platform=${platform}")
    platform_args+=("--platform=${platform}")
  done

  if [[ "${BUILDER}" == "nerdctl" ]]; then
    build_with_nerdctl "${extra_args[@]}"
    if [[ "${PUSH}" == "true" ]]; then
      for image in "${images[@]}"; do
        dry_run nerdctl push "${platform_args[@]}" "${image}"
      done
    fi
  else
    build_with_docker "${extra_args[@]}"
  fi
}

function build_with_docker() {
  local extra_args
  extra_args=("$@")
  if [[ "${PUSH}" == "true" ]]; then
    extra_args+=("--push")
  else
    extra_args+=("--load")
  fi
  dry_run docker buildx build \
    "${extra_args[@]}" \
    -f "${DOCKERFILE}" \
    .
}

function build_with_nerdctl() {
  local extra_args
  extra_args=("$@")
  dry_run nerdctl build \
    "${extra_args[@]}" \
    -f "${DOCKERFILE}" \
    .
}

args "$@"

cd "${ROOT_DIR}" && main
