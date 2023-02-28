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

DIR="$(realpath "${DIR}")"

ROOT_DIR="$(realpath "${DIR}/../..")"

VERSION="$("${ROOT_DIR}/hack/get-version.sh")"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

CASES_PATH="${DIR}/cases.txt"

function clear_testdata() {
  local kind="${1}"
  sed "s|${VERSION}|<VERSION>|g" |
    clear_testdata_goos "${kind}" |
    clear_testdata_goarch "${kind}"
}


function clear_testdata_goos() {
  local kind="${1}"
  if [[ "${kind}" == *"GOOS"* ]]; then
    sed "s|${GOOS}|<OS>|g"
  else
    cat
  fi
}

function clear_testdata_goarch() {
  local kind="${1}"
  if [[ "${kind}" == *"GOARCH"* ]]; then
    sed "s|${GOARCH}|<ARCH>|g"
  else
    cat
  fi
}

function main() {
  failed=()
  export DRY_RUN=true
  export IMAGE_PREFIX=image-prefix
  export PREFIX=prefix
  export STAGING_PREFIX=staging-prefix
  export NUMBER_SUPPORTED_KUBE_RELEASES="${NUMBER_SUPPORTED_KUBE_RELEASES:-3}"
  export BUILD_DATE=date


  while read -r line; do
    local args="${line}"
    local name="${args%% *}"
    args="${args#* }"
    local kind="${args%% *}"
    args="${args#* }"

    echo "------------------------------"
    echo "Testing ${name}, ${args}"
    want_file="${DIR}/testdata/${name}.txt"
    want="$(<"${want_file}")"

    IFS=' ' read -r -a argsArray <<<"${args}"

    got="$(make --no-print-directory -C "${ROOT_DIR}" "${argsArray[@]}" |clear_testdata "${kind}")"
    if [[ "${got}" != "${want}" ]]; then
      diff -u <(echo "${want}") <(echo "${got}")
      failed+=("${name}")
      if [[ "${UPDATE_DRY_RUN_TESTDATA}" == "true" ]]; then
        echo "${got}" >"${want_file}"
      else
        # prints the command to update the testdata
        echo "------------------------------"
        echo "cat <<ALL >${want_file}"
        echo "${got}"
        echo "ALL"
      fi
    fi
  done <"${CASES_PATH}"
  echo "------------------------------"

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

main
