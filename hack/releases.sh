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

ROOT_DIR="$(realpath "${DIR}/..")"

BUCKET=""
GH_RELEASE=""
IMAGE_PREFIX=""
BINARY_PREFIX=""
BINARY_NAME=""
VERSION=""
STAGING_PREFIX=""
DRY_RUN=false
PUSH=false
BINS=()
EXTRA_TAGS=()
PLATFORMS=()
LDFLAGS=()

function usage() {
  echo "Usage: ${0} [--help] [--bin <bin> ...] [--extra-tag <extra-tag> ...] [--platform <platform> ...] [--bucket <bucket>] [--image-prefix <image-prefix>] [--binary-prefix <binary-prefix>] [--binary-name <binary-name>] [--version <version>] [--staging-prefix <staging-prefix>] [--push] [--dry-run]"
  echo "  --bin <bin> is binary, is required"
  echo "  --extra-tag <extra-tag> is extra tag"
  echo "  --platform <platform> is multi-platform capable for binary"
  echo "  --bucket <bucket> is bucket to upload to"
  echo "  --gh-release <gh-release> is github release"
  echo "  --image-prefix <image-prefix> is kwok image prefix"
  echo "  --binary-prefix <binary-prefix> is kwok binary prefix"
  echo "  --binary-name <binary-name> is kwok binary name"
  echo "  --version <version> is version of binary"
  echo "  --staging-prefix <staging-prefix> is staging prefix for bucket"
  echo "  --push will push binary to bucket"
  echo "  --dry-run just show what would be done"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --bin | --bin=*)
      [[ "${arg#*=}" != "${arg}" ]] && BINS+=("${arg#*=}") || { BINS+=("${2}") && shift; }
      shift
      ;;
    --extra-tag | --extra-tag=*)
      [[ "${arg#*=}" != "${arg}" ]] && EXTRA_TAGS+=("${arg#*=}") || { EXTRA_TAGS+=("${2}") && shift; }
      shift
      ;;
    --platform | --platform=*)
      [[ "${arg#*=}" != "${arg}" ]] && PLATFORMS+=("${arg#*=}") || { PLATFORMS+=("${2}") && shift; }
      shift
      ;;
    --bucket | --bucket=*)
      [[ "${arg#*=}" != "${arg}" ]] && BUCKET="${arg#*=}" || { BUCKET="${2}" && shift; }
      shift
      ;;
    --gh-release | --gh-release=*)
      [[ "${arg#*=}" != "${arg}" ]] && GH_RELEASE="${arg#*=}" || { GH_RELEASE="${2}" && shift; }
      shift
      ;;
    --image-prefix | --image-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && IMAGE_PREFIX="${arg#*=}" || { IMAGE_PREFIX="${2}" && shift; }
      shift
      ;;
    --binary-prefix | --binary-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && BINARY_PREFIX="${arg#*=}" || { BINARY_PREFIX="${2}" && shift; }
      shift
      ;;
    --binary-name | --binary-name=*)
      [[ "${arg#*=}" != "${arg}" ]] && BINARY_NAME="${arg#*=}" || { BINARY_NAME="${2}" && shift; }
      shift
      ;;
    --version | --version=*)
      [[ "${arg#*=}" != "${arg}" ]] && VERSION="${arg#*=}" || { VERSION="${2}" && shift; }
      shift
      ;;
    --staging-prefix | --staging-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && STAGING_PREFIX="${arg#*=}" || { STAGING_PREFIX="${2}" && shift; }
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

  if [[ "${#BINS}" -eq 0 ]]; then
    echo "--bin is required"
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
  local os
  local dist
  local src
  local bin
  local tmp_bin
  local extra_args=()
  local prefix

  if [[ "${VERSION}" != "" ]]; then
    LDFLAGS+=("-X sigs.k8s.io/kwok/pkg/consts.Version=${VERSION}")
  fi
  if [[ "${IMAGE_PREFIX}" != "" ]]; then
    LDFLAGS+=("-X sigs.k8s.io/kwok/pkg/consts.ImagePrefix=${IMAGE_PREFIX}")
  fi
  if [[ "${BINARY_PREFIX}" != "" ]]; then
    LDFLAGS+=("-X sigs.k8s.io/kwok/pkg/consts.BinaryPrefix=${BINARY_PREFIX}")
  fi
  if [[ "${BINARY_NAME}" != "" ]]; then
    LDFLAGS+=("-X sigs.k8s.io/kwok/pkg/consts.BinaryName=${BINARY_NAME}")
  fi
  if [[ "${#LDFLAGS}" -gt 0 ]]; then
    extra_args+=("-ldflags" "'${LDFLAGS[*]}'")
  fi

  for platform in "${PLATFORMS[@]}"; do
    os="${platform%%/*}"
    for binary in "${BINS[@]}"; do
      bin="${binary}"
      if [[ "${os}" == "windows" ]]; then
        bin="${bin}.exe"
      fi
      dist="./bin/${platform}/${bin}"
      src="./cmd/${binary}"
      CGO_ENABLED=0 dry_run GOOS="${platform%%/*}" GOARCH="${platform##*/}" go build "${extra_args[@]}" -o "${dist}" "${src}"
      if [[ "${PUSH}" == "true" ]]; then
        if [[ "${BUCKET}" != "" ]]; then
          prefix="${BUCKET}/releases/"
          if [[ "${STAGING_PREFIX}" != "" ]]; then
            prefix="${BUCKET}/releases/${STAGING_PREFIX}-"
          fi
          dry_run gsutil cp -P "${dist}" "${prefix}${VERSION}/bin/${platform}/${bin}"
          if [[ "${#EXTRA_TAGS}" -ne 0 ]]; then
            for extra_tag in "${EXTRA_TAGS[@]}"; do
              dry_run gsutil cp -P "${dist}" "${prefix}${extra_tag}/bin/${platform}/${bin}"
            done
          fi
        fi
        if [[ "${GH_RELEASE}" != "" ]]; then
          tmp_bin="${binary}-${platform%%/*}-${platform##*/}"
          if [[ "${os}" == "windows" ]]; then
            tmp_bin="${tmp_bin}.exe"
          fi
          dry_run cp "${dist}" "${tmp_bin}"
          dry_run gh -R "${GH_RELEASE}" release upload "${VERSION}" "${tmp_bin}"
          if [[ "${#EXTRA_TAGS}" -ne 0 ]]; then
            for extra_tag in "${EXTRA_TAGS[@]}"; do
              dry_run gh -R "${GH_RELEASE}" release upload "${extra_tag}" "${tmp_bin}"
            done
          fi
        fi
      fi
    done
  done
}

args "$@"

cd "${ROOT_DIR}" && main
