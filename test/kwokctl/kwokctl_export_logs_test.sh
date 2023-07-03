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

source "${DIR}/suite.sh"

RELEASES=()

function usage() {
  echo "Usage: $0 <kube-version...>"
  echo "  <kube-version> is the version of kubernetes to test against."
}

function args() {
  if [[ $# -eq 0 ]]; then
    usage
    exit 1
  fi
  while [[ $# -gt 0 ]]; do
    RELEASES+=("${1}")
    shift
  done
}

function test_export_logs() {
  local name="${1}"
  local root_dir="${KWOK_LOGS_DIR}/kwok-${name}"

  if [[ "${SKIP_DRY_RUN}" != "true" ]]; then
    got="$(save_logs "${name}" --dry-run | clear_testdata "${name}")"
    want="$(<"${DIR}/testdata/${KWOK_RUNTIME}/export_logs.txt")"
    if [[ "${got}" != "${want}" ]]; then
      echo "------------------------------"
      diff -u <(echo "${want}") <(echo "${got}")
      echo "Error: dry run export logs failed"
      if [[ "${UPDATE_DRY_RUN_TESTDATA}" == "true" ]]; then
        echo "${got}" >"${DIR}/testdata/${KWOK_RUNTIME}/export_logs.txt"
      fi
      echo "------------------------------"
      echo "cat <<ALL >${DIR}/testdata/${KWOK_RUNTIME}/export_logs.txt"
      echo "${got}"
      echo "ALL"
      echo "------------------------------"
      return 1
    fi
  fi

  if ! save_logs "${name}"; then
    echo "Error: export logs failed"
    return 1
  fi

  if [[ ! -d "${root_dir}" ]]; then
    echo "Required directory ${root_dir} does not exist."
    return 1
  fi

  REQUIRED_FILES=(
    "kwok.yaml"
    "${KWOK_RUNTIME}-info.txt"
  )

  for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "${root_dir}/$file" ]; then
      echo "Required file $root_dir/$file does not exist."
      return 1
    fi
  done

  if [ ! -d "${root_dir}/components" ]; then
    echo "Required directory ${root_dir}/components does not exist."
    return 1
  fi

  LOG_FILES=(
    "audit.log"
    "etcd.log"
    "kube-apiserver.log"
    "kube-controller-manager.log"
    "kube-scheduler.log"
    "kwok-controller.log"
  )
  for file in "${LOG_FILES[@]}"; do
    if [[ ! -f "${root_dir}/components/${file}" ]]; then
      echo "Required file ${root_dir}/components/${file} does not exist."
      return 1
    fi
  done

  echo "Directory ${KWOK_LOGS_DIR} is correct."
}

function main() {
  local failed=()

  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing export logs on ${KWOK_RUNTIME} for ${release}"
    name="export-logs-cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --kube-audit-policy="${DIR}/audit-policy.yaml" -v=4
    test_export_logs "${name}" || failed+=("${name}_export_logs")
    delete_cluster "${name}"
  done

  if [[ "${#failed[@]}" -ne 0 ]]; then
    echo "------------------------------"
    echo "Error: Some tests failed"
    for test in "${failed[@]}"; do
      echo " - ${test}"
    done
    exit 1
  fi
}

args "$@"

main
