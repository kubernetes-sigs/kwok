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

COMMAND=()
if command -v yamllint; then
  COMMAND=(yamllint)
elif command -v "${ROOT_DIR}/venv/bin/yamllint"; then
  COMMAND=("${ROOT_DIR}/venv/bin/yamllint")
elif command -v python3; then
  python3 -m venv "${ROOT_DIR}/venv"
  source "${ROOT_DIR}/venv/bin/activate"
  pip install yamllint
  COMMAND=("${ROOT_DIR}/venv/bin/yamllint")
elif command -v docker; then
  COMMAND=(
    docker run
    --rm -i
    -v "${ROOT_DIR}:/workdir"
    -w "/workdir"
    --security-opt="label=disable"
    "docker.io/cytopia/yamllint:1.26@sha256:1bf8270a671a2e5f2fea8ac2e80164d627e0c5fa083759862bbde80628f942b2"
  )
else
  echo "WARNING: yamllint, python3 or docker not installed" >&2
  exit 1
fi

function check() {
  echo "Verify go format"
  mapfile -t findfiles < <(find . \( \
    -iname "*.yaml" \
    -o -iname "*.yml" \
    \) \
    -not \( \
    -path ./vendor/\* \
    -o -path ./demo/node_modules/\* \
    -o -path ./site/themes/\* \
    \))

  "${COMMAND[@]}" -s -c .yamllint.conf "${findfiles[@]}"
}

cd "${ROOT_DIR}" && check
