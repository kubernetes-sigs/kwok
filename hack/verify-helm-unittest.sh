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

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

HELM_UNITTEST_VERSION="1.2.1"

function runtime_os() {
  case "$(uname -s)" in
  Linux) echo linux ;;
  Darwin) echo macos ;;
  *)
    echo "Unsupported OS: $(uname -s)" >&2
    return 1
    ;;
  esac
}

function runtime_arch() {
  case "$(uname -m)" in
  x86_64 | amd64) echo amd64 ;;
  aarch64 | arm64) echo arm64 ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    return 1
    ;;
  esac
}

# The standalone untt binary does not need helm, which the verify image lacks.
function install_untt() {
  local os arch dir
  os="$(runtime_os)"
  arch="$(runtime_arch)"
  dir="${ROOT_DIR}/bin/helm-unittest-${HELM_UNITTEST_VERSION}"
  if [[ ! -x "${dir}/untt-${os}-${arch}" ]]; then
    mkdir -p "${dir}"
    curl -sSL "https://github.com/helm-unittest/helm-unittest/releases/download/v${HELM_UNITTEST_VERSION}/helm-unittest-${os}-${arch}-${HELM_UNITTEST_VERSION}.tgz" |
      tar -xz -C "${dir}" "untt-${os}-${arch}"
  fi
  echo "${dir}/untt-${os}-${arch}"
}

function check() {
  echo "Verify helm charts with helm-unittest"
  local untt
  untt="$(install_untt)"

  local charts=()
  for chart in charts/*/; do
    if [[ -d "${chart}tests" ]]; then
      charts+=("${chart}")
    fi
  done
  "${untt}" "${charts[@]}"
}

cd "${ROOT_DIR}" && check
