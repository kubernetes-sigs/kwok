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

show_info() {
  sleep 1

  # Print some useful information
  echo "###############################################################################"
  echo "> kubectl -s :${KWOK_KUBE_APISERVER_PORT} version"
  kwokctl kubectl version || true

  echo "###############################################################################"
  echo "# The following kubeconfig can be used to connect to the Kubernetes API server"
  cat <<EOF
apiVersion: v1
clusters:
- cluster:
    server: http://127.0.0.1:${KWOK_KUBE_APISERVER_PORT}
  name: kwok
contexts:
- context:
    cluster: kwok
  name: kwok
current-context: kwok
kind: Config
preferences: {}
users: null
EOF

  echo "###############################################################################"
  echo "> kubectl -s :${KWOK_KUBE_APISERVER_PORT} get ns"
  kwokctl kubectl get ns || true

  echo "###############################################################################"
  echo "# The above example works if your host's port is the same as the container's,"
  echo "# otherwise, change it to your host's port"
}

# Create a cluster
KWOK_KUBE_APISERVER_PORT=0 kwokctl create cluster "$@" || exit 1

show_info &

# Start a proxy to the Kubernetes API server
kwokctl kubectl proxy --port="${KWOK_KUBE_APISERVER_PORT}" --accept-hosts='^*$' --address="0.0.0.0"
