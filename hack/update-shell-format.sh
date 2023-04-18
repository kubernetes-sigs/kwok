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

ROOT_DIR="$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)"

all_shell_scripts=()
while IFS=$'\n' read -r script; do
  git check-ignore -q "$script" || all_shell_scripts+=("$script")
done < <(find . -name "*.sh" \
  -not \( \
  -path ./.git\* -o \
  -path ./vendor\* \
  \))

COMMAND=()
if command -v shfmt; then
  COMMAND=(shfmt)
elif command -v "${ROOT_DIR}/bin/shfmt"; then
  COMMAND=("${ROOT_DIR}/bin/shfmt")
elif [[ "$(uname -s)" == "Linux" ]] && [[ "$(uname -m)" == "x86_64" ]]; then
  wget "https://github.com/mvdan/sh/releases/download/v3.6.0/shfmt_v3.6.0_linux_amd64"
  mkdir -p "${ROOT_DIR}"/bin
  mv "shfmt_v3.6.0_linux_amd64" "${ROOT_DIR}/bin/shfmt"
  chmod +x "${ROOT_DIR}/bin/shfmt"
  COMMAND=("${ROOT_DIR}/bin/shfmt")
elif command -v docker; then
  COMMAND=(
    docker run
    --rm
    -v "${ROOT_DIR}:${ROOT_DIR}"
    -w "${ROOT_DIR}"
    docker.io/mvdan/shfmt:v3.6.0-alpine
    shfmt
  )
else
  echo "WARNING: shfmt or docker not installed" >&2
  exit 1
fi

cd "${ROOT_DIR}" && "${COMMAND[@]}" -w -i=2 "${all_shell_scripts[@]}"
