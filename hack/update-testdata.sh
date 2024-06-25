#!/usr/bin/env bash
# Copyright 2024 The Kubernetes Authors.
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

TESTDATA_DIR="${ROOT_DIR}/test/e2e/kwokctl/dryrun"

# shellcheck source=/dev/null
source "${ROOT_DIR}/test/kwokctl/helper.sh"
# shellcheck source=/dev/null
source "${ROOT_DIR}/test/kwokctl/suite.sh"

function update() {
  local runtimes=("${@}")
  local name
  local got

  for runtime in "${runtimes[@]}"; do
    name="dryrun-cluster-${runtime}"
    echo "------------------------------"
    echo "Updating dryrun on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster "${name}" "" \
      --kube-authorization=false \
      --dry-run | clear_testdata "${name}")"
    echo "${got}" >"${TESTDATA_DIR}/testdata/${runtime}/create_cluster.txt"
    echo "------------------------------"
    echo "Updating dryrun with extra on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster <"${ROOT_DIR}/test/kwokctl/testdata/extra.yaml" "${name}" "" --config - --dry-run | clear_testdata "${name}")"
    echo "${got}" >"${TESTDATA_DIR}/testdata/${runtime}/create_cluster_with_extra.txt"
    echo "------------------------------"
    echo "Updating dryrun with verbosity on runtime ${runtime}"
    got="$(KWOK_RUNTIME="${runtime}" create_cluster "${name}" "" -v=debug \
      --prometheus-port 9090 \
      --jaeger-port 16686 \
      --dashboard-port 8000 \
      --kube-audit-policy "${DIR}/audit-policy.yaml" \
      --kube-scheduler-config "${DIR}/scheduler-config.yaml" \
      --enable-metrics-server \
      --dry-run | clear_testdata "${name}")"
    echo "${got}" >"${TESTDATA_DIR}/testdata/${runtime}/create_cluster_with_verbosity.txt"
  done

}

requirements_for_binary

cd "${ROOT_DIR}" && update binary docker podman nerdctl kind kind-podman
