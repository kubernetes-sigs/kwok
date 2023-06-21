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

# inputs are:
# - LAST_VERSION_TAG -- This is the version to get commits since
#    like: LAST_VERSION_TAG="v0.1.0"
# - GH_TOKEN -- used to avoid hitting API rate limits
ORG="${ORG:-kubernetes-sigs}"
REPO="${REPO:-kwok}"

# https://github.com/kubernetes/release/blob/master/cmd/release-notes/README.md

mapfile -t hashs < <(git log --format="%H" "${LAST_VERSION_TAG:?}..")

tmp_file=$(mktemp)

GITHUB_TOKEN=${GH_TOKEN:-} go run k8s.io/release/cmd/release-notes@v0.15.1 \
  --org "${ORG}" \
  --repo "${REPO}" \
  --branch main \
  --start-sha "${hashs[-1]}" \
  --end-sha "${hashs[0]}" \
  --markdown-links=false \
  --required-author "" \
  --output "${tmp_file}"

cat "${tmp_file}"
