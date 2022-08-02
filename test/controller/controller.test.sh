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

KIND_NODE_IMAGE=docker.io/kindest/node:v1.24.2
CLUSTER_NAME=kwok-controller-test
IMAGE=kwok-controller:test

"${DIR}"/../../images/kwok/build.sh --tag "${IMAGE}"

kind create cluster --name="${CLUSTER_NAME}" --image="${KIND_NODE_IMAGE}"

kind load docker-image --name="${CLUSTER_NAME}" ${IMAGE}

kubectl kustomize "${DIR}" | kubectl apply -f -

# Waiting for the cluster to be ready

for i in {1..30}; do
  if [[ ! "$(kubectl get node fake-node)" =~ "Ready" ]]; then
    echo "Waiting for fake-node to be ready..."
    sleep 1
  else
    break
  fi
done

# Check for normal heartbeat
if [[ ! "$(kubectl get node fake-node)" =~ "Ready" ]]; then
  echo "fake-node is not ready"
  kubectl get node fake-node
  exit 1
fi

# Check for the Pod is running

for i in {1..30}; do
  if [[ ! "$(kubectl get pod | grep Running | wc -l)" -eq 5 ]]; then
    echo "Waiting all pods to be running..."
    sleep 1
  else
    break
  fi
done

if [[ ! "$(kubectl get pod | grep Running | wc -l)" -eq 5 ]]; then
  echo "Not all pods are running"
  kubectl get pod -o wide
  exit 1
fi

# Check for the status of the Node is modified by Kubectl
kubectl annotate node fake-node --overwrite kwok.x-k8s.io/status=custom
kubectl patch node fake-node --subresource=status -p '{"status":{"nodeInfo":{"kubeletVersion":"fake-new"}}}'

sleep 2

if [[ ! "$(kubectl get node fake-node)" =~ "fake-new" ]]; then
  echo "fake-node is not updated"
  kubectl get node fake-node
  exit 1
fi

# Check for the status of the Pod is modified by Kubectl
FIRST_POD=$(kubectl get pod | grep Running | head -n 1 | awk '{print $1}')

kubectl annotate pod "${FIRST_POD}" --overwrite kwok.x-k8s.io/status=custom
kubectl patch pod "${FIRST_POD}" --subresource=status -p '{"status":{"podIP":"192.168.0.1"}}'

sleep 2

if [[ ! "$(kubectl get pod "${FIRST_POD}" -o wide)" =~ "192.168.0.1" ]]; then
  echo "fake-pod is not updated"
  kubectl get pod "${FIRST_POD}" -o wide
  exit 1
fi

# Cleanup

kind delete cluster --name="${CLUSTER_NAME}"
