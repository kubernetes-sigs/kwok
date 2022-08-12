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

args=()

KUBE_VERSION="v1.24.2"
while [[ $# -gt 0 ]]; do
  arg="$1"
  case "${arg}" in
  --kube-version | --kube-version=*)
    [[ "${key#*=}" != "${key}" ]] && KUBE_VERSION="${key#*=}" || { KUBE_VERSION="${2}" && shift; }
    shift
    ;;
  *)
    args+=("${arg}")
    ;;
  esac
  shift
done

docker build \
  "${args[@]}" \
  --build-arg kube_version="${KUBE_VERSION}" \
  -f "${DIR}/Dockerfile" \
  "${DIR}/../.."
