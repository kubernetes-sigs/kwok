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

if [[ "${GIT_TAG}" == "" ]]; then
  GIT_TAG=$(git describe --tags --dirty --always)
fi

if [[ ! "${GIT_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  GIT_TAG="latest"
fi

echo "${GIT_TAG}"
