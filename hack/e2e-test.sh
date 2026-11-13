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

test_dir=$(realpath "${DIR}"/../test)

TARGETS=()
SKIPS=()
RETRY=0

function filter_skip() {
  cat | while read -r line; do
    skip=false
    for s in "${SKIPS[@]}"; do
      if [[ "${line}" =~ ${s} ]]; then
        skip=true
        break
      fi
    done
    if [[ "${skip}" == false ]]; then
      echo "${line}"
    fi
  done
}

function shell_cases() {
  find "${test_dir}" -name '*.test.sh' |
    sed "s#^${test_dir}/##g" |
    sed "s#.test.sh\$##g"
}

function e2e_cases() {
  find test/e2e -type f -name 'main_test.go' |
    sed 's|/main_test.go$||g' |
    sed 's|^test/||g'
}

function e2e_option_cases() {
  local files
  local cases
  files=$(find test/e2e -type f -name '*_test.go' -not -name 'main_test.go')
  for f in ${files}; do
    cases=$(grep "^func Test" <"${f}" | sed 's|(.*||g' | sed 's|^func Test||g')
    f="${f%\/*}"
    f="${f#*\/}"
    for c in ${cases}; do
      echo "${f}@${c}"
    done
  done
}

function all_cases() {
  shell_cases | filter_skip
  e2e_cases | filter_skip
}

function usage() {
  echo "Usage: ${0} [cases...] [--retry=N] [--help]"
  echo "  Empty argument will run all cases."
  echo "  --retry=N  Re-run each failed case up to N more times (default 0)."
  echo "  CASES:"
  for c in $(shell_cases | filter_skip); do
    echo "    ${c}"
  done
  for c in $(e2e_cases | filter_skip); do
    echo "    ${c}"
  done
  for c in $(e2e_option_cases | filter_skip); do
    echo "    ${c}"
  done
}

function args() {
  if [[ "${#}" -ne 0 ]]; then
    while [[ $# -gt 0 ]]; do
      arg="$1"
      case ${arg} in
      --help)
        usage
        exit 0
        ;;
      --skip | --skip=*)
        [[ "${arg#*=}" != "${arg}" ]] && SKIPS+=("${arg#*=}") || { SKIPS+=("${2}") && shift; } || :
        shift
        ;;
      --retry | --retry=*)
        if [[ "${arg}" == "--retry" ]]; then
          RETRY="${2}"
          shift
        else
          RETRY="${arg#*=}"
        fi
        shift
        if [[ ! "${RETRY}" =~ ^[0-9]+$ ]]; then
          echo "Error: Invalid value for --retry: ${RETRY}"
          usage
          exit 1
        fi
        # Force base 10 so leading zeros are not read as octal.
        RETRY=$((10#${RETRY}))
        ;;
      -*)
        echo "Error: Unknown argument: ${arg}"
        usage
        exit 1
        ;;
      *)
        TARGETS+=("${arg}")
        shift
        ;;
      esac
    done
  fi
  if [[ "${#TARGETS[@]}" == 0 ]]; then
    mapfile -t TARGETS < <(all_cases)
  fi
}

function run_target() {
  local target="${1}"
  local test_case
  local test_path
  if [[ "${target}" == "e2e/"* ]]; then
    if [[ "${target}" == *"@"* ]]; then
      test_case="${target##*@}"
      test_path="${target%@*}"
      go test -timeout=1h -v -test.v "sigs.k8s.io/kwok/test/${test_path}" -test.run "^Test${test_case}\$" -args --v=6
    else
      go test -timeout=1h -v -test.v "sigs.k8s.io/kwok/test/${target}" -args --v=6
    fi
  else
    "${test_dir}/${target}.test.sh"
  fi
}

function main() {
  local failed=()
  local attempt
  local attempts=$((RETRY + 1))
  for target in "${TARGETS[@]}"; do
    echo "================================================================================"
    if [[ "${target}" != "e2e/"* ]]; then
      target="${target%.test.sh}"
      test="${test_dir}/${target}.test.sh"
      if [[ ! -x "${test}" ]]; then
        echo "Error: Test ${test} not found."
        failed+=("${test}")
        continue
      fi
    fi

    attempt=1
    while true; do
      if [[ "${RETRY}" -eq 0 ]]; then
        echo "Testing ${target}..."
      else
        echo "Testing ${target} (attempt ${attempt} of ${attempts})..."
      fi
      if run_target "${target}"; then
        echo "------------------------------"
        echo "Test ${target} passed."
        if [[ "${attempt}" -gt 1 ]]; then
          echo "Test ${target} passed on retry (attempt ${attempt} of ${attempts})."
        fi
        break
      fi
      echo "------------------------------"
      if [[ "${attempt}" -ge "${attempts}" ]]; then
        failed+=("${target}")
        echo "Test ${target} failed."
        break
      fi
      echo "Test ${target} failed (attempt ${attempt} of ${attempts}), retrying..."
      attempt=$((attempt + 1))
    done
  done
  echo "================================================================================"

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

args "$@"

main
