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

function filter_e2e_option_cases() {
  jq -r '. | select( .Action == "pass" ) | select( has("Test") ) | select( .Test | contains("/") | not ) | .Package + "@" + .Test' |
    sed 's|^sigs.k8s.io/kwok/test/||g' |
    sed 's|@Test|@|g'
}

function filter_e2e_cases() {
  jq -r '. | select( .Action == "pass" ) | select( .Test == null ) | .Package' |
    sed 's|^sigs.k8s.io/kwok/test/||g'
}

function shell_cases() {
  find "${test_dir}" -name '*.test.sh' |
    sed "s#^${test_dir}/##g" |
    sed "s#.test.sh\$##g" |
    filter_skip
}

function e2e_cases() {
  go test -json -timeout 10s sigs.k8s.io/kwok/test/e2e/kwok/... -args --dry-run |
    filter_e2e_cases |
    filter_skip
  go test -json -timeout 10s sigs.k8s.io/kwok/test/e2e/kwokctl/... -args --dry-run |
    filter_e2e_cases |
    filter_skip
}

function e2e_option_cases() {
  go test -json -timeout 10s sigs.k8s.io/kwok/test/e2e/kwok/... -args --dry-run |
    filter_e2e_option_cases |
    filter_skip
  go test -json -timeout 10s sigs.k8s.io/kwok/test/e2e/kwokctl/... -args --dry-run |
    filter_e2e_option_cases |
    filter_skip
}

function all_cases() {
  shell_cases
  e2e_cases
}

function usage() {
  echo "Usage: ${0} [cases...] [--help]"
  echo "  Empty argument will run all cases."
  echo "  CASES:"
  for c in $(shell_cases); do
    echo "    ${c}"
  done
  for c in $(e2e_cases); do
    echo "    ${c}"
  done
  for c in $(e2e_option_cases); do
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

function main() {
  local failed=()
  local test_case
  local test_path
  for target in "${TARGETS[@]}"; do
    echo "================================================================================"
    if [[ "${target}" == "e2e/"* ]]; then
      echo "Testing ${target}..."
      if [[ "${target}" == *"@"* ]]; then
        test_case="${target##*@}"
        test_path="${target%@*}"
        if ! go test -timeout=1h -v -test.v "sigs.k8s.io/kwok/test/${test_path}" -test.run "^Test${test_case}\$" -args --v=6; then
          failed+=("${target}")
          echo "------------------------------"
          echo "Test ${target} failed."
        else
          echo "------------------------------"
          echo "Test ${target} passed."
        fi
        continue
      fi

      if ! go test -timeout=1h -v -test.v "sigs.k8s.io/kwok/test/${target}" -args --v=6; then
        failed+=("${target}")
        echo "------------------------------"
        echo "Test ${target} failed."
      else
        echo "------------------------------"
        echo "Test ${target} passed."
      fi
      continue
    fi

    target="${target%.test.sh}"
    test="${test_dir}/${target}.test.sh"
    if [[ ! -x "${test}" ]]; then
      echo "Error: Test ${test} not found."
      failed+=("${test}")
      continue
    fi

    echo "Testing ${target}..."
    if ! "${test_dir}/${target}.test.sh"; then
      failed+=("${target}")
      echo "------------------------------"
      echo "Test ${target} failed."
    else
      echo "------------------------------"
      echo "Test ${target} passed."
    fi
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
