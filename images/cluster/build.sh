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
KUBE_VERSIONS=()

function usage() {
  echo "Usage: ${0} [--help] [--version <version>] [--kube-version <kube-version> ...] [--image <image> ...] [--extra-tag <extra-tag> ...] [--staging-prefix <staging-prefix>] [--platform <platform> ...] [--push] [--dry-run]"
  echo "  --version <version> is kwok version, is required"
  echo "  --kube-version <kube-version> is kubernetes version, is required"
  echo "  --image <image> is image, is required"
  echo "  --extra-tag <extra-tag> is extra tag"
  echo "  --staging-prefix <staging-prefix> is staging prefix for tag"
  echo "  --platform <platform> is multi-platform capable for image"
  echo "  --push will push image to registry"
  echo "  --dry-run just show what would be done"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --kube-version | --kube-version=*)
      [[ "${arg#*=}" != "${arg}" ]] && KUBE_VERSIONS+=("${arg#*=}") || { KUBE_VERSIONS+=("${2}") && shift; }
      shift
      ;;
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

  if [[ "${#KUBE_VERSIONS}" -eq 0 ]]; then
    echo "--kube-version is required"
    usage
    exit 1
  fi

  if [[ "${#IMAGES}" -eq 0 ]]; then
    echo "--image is required"
    usage
    exit 1
  fi

  if [[ "${#PLATFORMS}" -eq 0 ]]; then
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
  local suffix
  local tag

  for kube_version in "${KUBE_VERSIONS[@]}"; do
    extra_args=(
      "--build-arg=kube_version=${kube_version}"
    )

    suffix="-k8s.${kube_version}"

    for image in "${IMAGES[@]}"; do
      tag="${VERSION}${suffix}"
      if [[ "${STAGING_PREFIX}" != "" ]]; then
        tag="${STAGING_PREFIX}-${VERSION}${suffix}"
      fi
      extra_args+=(
        "--tag=${image}:${tag}"
      )
      if [[ "${#EXTRA_TAGS[@]}" -ne 0 ]]; then
        for extra_tag in "${EXTRA_TAGS[@]}"; do
          tag="${extra_tag}${suffix}"
          if [[ "${STAGING_PREFIX}" != "" ]]; then
            tag="${STAGING_PREFIX}-${extra_tag}${suffix}"
          fi
          extra_args+=(
            "--tag=${image}:${tag}"
          )
        done
      fi
    done

    for platform in "${PLATFORMS[@]}"; do
      extra_args+=(
        "--platform=${platform}"
      )
    done
    if [[ "${PUSH}" == "true" ]]; then
      extra_args+=("--push")
    else
      extra_args+=("--load")
    fi
    dry_run docker buildx build \
      "${extra_args[@]}" \
      -f "${DOCKERFILE}" \
      .
  done
}

args "$@"

cd "${ROOT_DIR}" && main
