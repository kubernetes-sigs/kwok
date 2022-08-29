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

# The value passed by gcr's cloudbuild will have this prefix by default
# https://github.com/kubernetes/k8s.io/blob/aa5a1f164aece8f116196c40ac7b937be479cd41/images/codesearch/cs-fetch-repos/Makefile#L19
if [[ "${GIT_TAG}" =~ ^v[0-9]{8}- ]]; then
  # Remove prefix for released version
  if [[ "${GIT_TAG:10}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-alpha\. ]]; then
    exit 0
  fi
  if [[ "${GIT_TAG:10}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-beta\. ]]; then
    exit 0
  fi
  if [[ "${GIT_TAG:10}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-rc\. ]]; then
    exit 0
  fi
  if [[ "${GIT_TAG:10}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    exit 0
  fi
  echo "${GIT_TAG:0:9}"
fi
