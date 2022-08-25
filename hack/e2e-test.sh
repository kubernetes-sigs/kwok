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

targets=()

function all_cases() {
  find "${test_dir}" -name '*.test.sh' | sed "s#^${test_dir}/##g" | sed "s#.test.sh\$##g" | sort
}

function usage() {
  echo "Usage: ${0} [cases...] [--help]"
  echo "  Empty argument will run all cases."
  echo "  CASES:"
  for c in $(all_cases); do
    echo "    ${c}"
  done
}

function args() {
  if [[ "${#}" -ne 0 ]]; then
    for arg in "${@}"; do
      case ${arg} in
      --help)
        usage
        exit 0
        ;;
      -*)
        echo "Error: Unknown argument: ${arg}"
        usage
        exit 1
        ;;
      *)
        targets+=("${arg}")
        ;;
      esac
    done
  else
    targets=(
      $(all_cases)
    )
  fi
}

function main() {
  local failed=()
  for target in "${targets[@]}"; do
    echo "================================================================================"
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
