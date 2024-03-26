#!/bin/sh
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

export KWOK_KUBE_APISERVER_INSECURE_PORT="${KWOK_KUBE_APISERVER_INSECURE_PORT:-${KWOK_KUBE_APISERVER_PORT}}"
export KWOK_KUBE_APISERVER_PORT=0

kwokctl create cluster || {
  echo "Failed to create cluster"
  exit 1
}

catch_exit() {
  kwokctl stop cluster || true
  exit 0
}

keep_alive() {
  while true; do
    if ! output="$(kwokctl start cluster 2>&1)"; then
      echo "Failed to start cluster: ${output}"
    fi
    sleep 10
  done
}

trap catch_exit INT TERM

keep_alive
