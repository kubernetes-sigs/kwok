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

source "${DIR}/helper.sh"
source "${DIR}/suite.sh"

function main() {
  local runtimes=("${@}")
  local failed=()
  local name
  local got

  for runtime in "${runtimes[@]}"; do
    name="dryrun-cluster-${runtime}"
    echo "------------------------------"
    echo "Testing dryrun on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster "${name}" "" --dry-run | clear_testdata "${name}")"
    want="$(<"${DIR}/testdata/${runtime}/create_cluster.txt")"
    if [[ "${got}" != "${want}" ]]; then
      echo "------------------------------"
      diff -u <(echo "${want}") <(echo "${got}")
      failed+=("create-cluster-runtime-${runtime}-dry-run")
      if [[ "${UPDATE_DRY_RUN_TESTDATA}" == "true" ]]; then
        echo "${got}" >"${DIR}/testdata/${runtime}/create_cluster.txt"
      fi
      echo "------------------------------"
      echo "cat <<ALL >${DIR}/testdata/${runtime}/create_cluster.txt"
      echo "${got}"
      echo "ALL"
      echo "------------------------------"
    fi
    echo "------------------------------"
    echo "Testing dryrun with extra on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster <"${DIR}/testdata/extra.yaml" "${name}" "" --config - --dry-run | clear_testdata "${name}")"
    want="$(<"${DIR}/testdata/${runtime}/create_cluster_with_extra.txt")"
    if [[ "${got}" != "${want}" ]]; then
      echo "------------------------------"
      diff -u <(echo "${want}") <(echo "${got}")
      failed+=("create-cluster-${runtime}-with-extra-dry-run")
      if [[ "${UPDATE_DRY_RUN_TESTDATA}" == "true" ]]; then
        echo "${got}" >"${DIR}/testdata/${runtime}/create_cluster_with_extra.txt"
      fi
      echo "------------------------------"
      echo "cat <<ALL >${DIR}/testdata/${runtime}/create_cluster_with_extra.txt"
      echo "${got}"
      echo "ALL"
      echo "------------------------------"
    fi
    echo "------------------------------"
    echo "Testing dryrun with verbosity on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster "${name}" "" -v=debug --enable-metrics-server --prometheus-port 9090 --jaeger-port 16686 --dashboard-port 8000 --dry-run | clear_testdata "${name}")"
    want="$(<"${DIR}/testdata/${runtime}/create_cluster_with_verbosity.txt")"
    if [[ "${got}" != "${want}" ]]; then
      echo "------------------------------"
      diff -u <(echo "${want}") <(echo "${got}")
      failed+=("create-cluster-${runtime}-with-verbosity-dry-run")
      if [[ "${UPDATE_DRY_RUN_TESTDATA}" == "true" ]]; then
        echo "${got}" >"${DIR}/testdata/${runtime}/create_cluster_with_verbosity.txt"
      fi
      echo "------------------------------"
      echo "cat <<ALL >${DIR}/testdata/${runtime}/create_cluster_with_verbosity.txt"
      echo "${got}"
      echo "ALL"
      echo "------------------------------"
    fi
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

requirements_for_binary

main binary docker podman nerdctl kind kind-podman
