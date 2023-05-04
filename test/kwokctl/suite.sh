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

function show_all() {
  for name in $(kwokctl get clusters); do
    show_info "${name}"
  done
}

function show_info() {
  local name="${1}"
  echo
  echo kwokctl --name="${name}" kubectl get pod -o wide --all-namespaces
  kwokctl --name="${name}" kubectl get pod -o wide --all-namespaces
  echo
  echo kwokctl --name="${name}" logs etcd
  kwokctl --name="${name}" logs etcd
  echo
  echo kwokctl --name="${name}" logs kube-apiserver
  kwokctl --name="${name}" logs kube-apiserver
  echo
  echo kwokctl --name="${name}" logs kube-controller-manager
  kwokctl --name="${name}" logs kube-controller-manager
  echo
  echo kwokctl --name="${name}" logs kube-scheduler
  kwokctl --name="${name}" logs kube-scheduler
  echo
  echo kwokctl --name="${name}" logs kwok-controller
  kwokctl --name="${name}" logs kwok-controller
  echo
}

function create_cluster() {
  local name="${1}"
  local release="${2}"
  shift 2

  if ! KWOK_KUBE_VERSION="${release}" kwokctl \
         create cluster \
         --name "${name}" \
         --timeout 30m \
         --wait 30m \
         --quiet-pull \
         "$@";
  then
    echo "Error: Cluster ${name} creation failed"
    show_all
    exit 1
  fi
}

function delete_cluster() {
  local name="${1}"

  if ! kwokctl delete cluster --name "${name}";
  then
    echo "Error: Cluster ${name} deletion failed"
    exit 1
  fi
}

function child_timeout() {
  local to="${1}"
  shift
  "${@}" &
  local wp=$!
  local start=0
  while kill -0 "${wp}" 2>/dev/null; do
    if [[ "${start}" -ge "${to}" ]]; then
      kill "${wp}"
      echo "Error: Timeout ${to}s" >&2
      return 1
    fi
    ((start++))
    sleep 1
  done
  echo "Took ${start}s" >&2
}
