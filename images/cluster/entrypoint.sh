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

export ALIVE=true

catch_exit() {
  ALIVE=false
  kwokctl stop cluster || true
  exit 0
}

keep_alive() {
  while [ "${ALIVE}" = "true" ]; do
    if ! output="$(kwokctl start cluster 2>&1)"; then
      echo "Failed to start cluster: ${output}"
    fi
    sleep 10
  done
}

show_info() {
  echo "Start cluster and keep alive at 0.0.0.0:${KWOK_KUBE_APISERVER_INSECURE_PORT}"
  echo "See more https://kwok.sigs.k8s.io/docs/user/all-in-one-image/"
}

trap catch_exit EXIT

kwokctl create cluster "$@" || exit 1

show_info

keep_alive
