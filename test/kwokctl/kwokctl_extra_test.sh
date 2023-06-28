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

EXTRASDIR="./extras"

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

function test_prometheus() {
  local targets
  for ((i = 0; i < 120; i++)); do
    targets="$(curl -s http://127.0.0.1:9090/api/v1/targets)"
    if [[ "$(echo "${targets}" | grep -o '"health":"up"' | wc -l)" -ge 6 ]]; then
      break
    fi
    sleep 1
  done

  if ! [[ "$(echo "${targets}" | grep -o '"health":"up"' | wc -l)" -ge 6 ]]; then
    echo "Error: metrics is not health"
    echo curl -s http://127.0.0.1:9090/api/v1/targets
    echo "${targets}"
    return 1
  fi
}

function prepare_mount_dirs() {
  mkdir "${EXTRASDIR}/apiserver"
  mkdir "${EXTRASDIR}/controller-manager"
  mkdir "${EXTRASDIR}/scheduler"
  mkdir "${EXTRASDIR}/controller"
  mkdir "${EXTRASDIR}/etcd"
  mkdir "${EXTRASDIR}/prometheus"
}

function main() {
  local failed=()
  local name

  mkdir -p "${EXTRASDIR}"
  prepare_mount_dirs
  for release in "${RELEASES[@]}"; do
    echo "------------------------------"
    echo "Testing extra on ${KWOK_RUNTIME} for ${release}"
    name="cluster-${KWOK_RUNTIME}-${release//./-}"
    create_cluster "${name}" "${release}" --config - <<EOF
apiVersion: config.kwok.x-k8s.io/v1alpha1
kind: KwokctlConfiguration
options:
  prometheusPort: 9090
componentsPatches:
- name: kube-apiserver
  extraArgs:
  - key: v
    value: "5"
  extraVolumes:
  - name: tmp-apiserver
    hostPath: ./extras/apiserver
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
- name: kube-controller-manager
  extraArgs:
  - key: v
    value: "5"
  extraVolumes:
  - name: tmp-controller-manager
    hostPath: ./extras/controller-manager
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
- name: kube-scheduler
  extraArgs:
  - key: v
    value: "5"
  extraVolumes:
  - name: tmp-scheduler
    hostPath: ./extras/scheduler
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
- name: kwok-controller
  extraArgs:
  - key: v
    value: "-4"
  extraVolumes:
  - name: tmp-controller
    hostPath: ./extras/controller
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
- name: etcd
  extraArgs:
  - key: log-level
    value: "debug"
  extraVolumes:
  - name: tmp-etcd
    hostPath: ./extras/etcd
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
- name: prometheus
  extraArgs:
  - key: log.level
    value: "debug"
  extraVolumes:
  - name: tmp-prometheus
    hostPath: ./extras/prometheus
    mountPath: /extras/tmp
    readOnly: false
    pathType: DirectoryOrCreate
EOF
    test_prometheus || failed+=("prometheus_${name}")
    delete_cluster "${name}"
  done
  echo "------------------------------"
  rm -rf "${EXTRASDIR}"

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
