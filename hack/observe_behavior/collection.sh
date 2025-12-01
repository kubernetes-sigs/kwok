#!/usr/bin/env bash
# Copyright 2025 The Kubernetes Authors.
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

rm -f logs/*

kind create cluster --config kind-config.yaml

sleep 5

kubectl create -f test-pod.yaml
kubectl wait --for=condition=Ready pod/test-pod --timeout=60s

sleep 5

kubectl delete -f test-pod.yaml
kubectl wait --for=delete pod/test-pod --timeout=60s


sleep 5

kind delete cluster

cat ./logs/kube-apiserver-audit.log | \
    grep test-pod | \
    go run audit-format.go > ./logs/audit-log.yaml
