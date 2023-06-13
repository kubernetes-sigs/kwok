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

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

SHELLCHECK_VERSION="0.9.0"

# disabled lints
disabled=(
  # this lint disallows non-constant source, which we use extensively without
  # any known bugs
  1090
  # this lint warns when shellcheck cannot find a sourced file
  # this wouldn't be a bad idea to warn on, but it fails on lots of path
  # dependent sourcing, so just disable enforcing it
  1091
  # this lint prefers command -v to which, they are not the same
  2230
)
# comma separate for passing to shellcheck
join_by() {
  local IFS="$1"
  shift
  echo "$*"
}
SHELLCHECK_DISABLED="$(join_by , "${disabled[@]}")"
readonly SHELLCHECK_DISABLED

mapfile -t findfiles < <(find . \( \
  -iname "*.sh" \
  \) \
  -not \( \
  -path ./vendor/\* \
  -o -path ./demo/node_modules/\* \
  -o -path ./site/themes/\* \
  \))

SHELLCHECK_OPTIONS=(
  "--external-sources"
  "--exclude=${SHELLCHECK_DISABLED}"
  "--color=always"
)

HAVE_SHELLCHECK=false
if command -v shellcheck; then
  detected_version="$(shellcheck --version | grep 'version: .*')"
  if [[ "${detected_version}" == "version: ${SHELLCHECK_VERSION}" ]]; then
    HAVE_SHELLCHECK=true
  fi
fi

COMMAND=()
if ${HAVE_SHELLCHECK}; then
  COMMAND=(shellcheck)
elif command -v "${ROOT_DIR}/bin/shellcheck"; then
  COMMAND=("${ROOT_DIR}/bin/shellcheck")
elif [[ "$(uname -s)" == "Linux" ]] && [[ "$(uname -m)" == "x86_64" ]]; then
  wget -qO- "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION?}/shellcheck-v${SHELLCHECK_VERSION?}.linux.x86_64.tar.xz" | tar -xJv
  mkdir -p "${ROOT_DIR}"/bin
  mv "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" "${ROOT_DIR}/bin/"
  COMMAND=("${ROOT_DIR}/bin/shellcheck")
elif command -v docker; then
  COMMAND=(
    docker run
    --rm
    -v "${ROOT_DIR}:${ROOT_DIR}"
    -w "${ROOT_DIR}"
    docker.io/koalaman/shellcheck-alpine:v0.9.0@sha256:e19ed93c22423970d56568e171b4512c9244fc75dd9114045016b4a0073ac4b7
    shellcheck
  )
else
  echo "WARNING: shellcheck or docker not installed" >&2
  exit 1
fi

cd "${ROOT_DIR}" && "${COMMAND[@]}" "${SHELLCHECK_OPTIONS[@]}" "${findfiles[@]}"
