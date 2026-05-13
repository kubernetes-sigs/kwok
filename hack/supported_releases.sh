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

ROOT_DIR="$(realpath "${DIR}/..")"

record="${ROOT_DIR}/supported_releases.txt"

# Get latest releases from GitHub
function latest_releases() {
  gh release -R kubernetes/kubernetes list --exclude-drafts --exclude-pre-releases --json=tagName --jq '.[].tagName' --limit 100
}

# Get historical supported releases
function record_releases() {
  cat "${record}"
}

# Pick the latest of each releases
function bump() {
  local raw_all_releases
  raw_all_releases="$*"
  declare -A release_map=()
  for release in ${raw_all_releases}; do
    release="${release//v/}"
    minor="${release%.*}"
    patch="${release##*.}"

    if [[ "${release_map["${minor}"]}" -le "${patch}" ]]; then
      release_map["${minor}"]="${patch}"
    fi
  done

  for r in "${!release_map[@]}"; do
    echo "${r}.${release_map[${r}]}"
  done
}

function main() {
  local record_data
  local latest_data
  latest_data="$(latest_releases)"

  if [[ "${latest_data}" == "" ]]; then
    echo "Failed get latest releases"
    return 1
  fi

  record_data="$(record_releases)"
  out="$(bump "${record_data}" "${latest_data}" | sort --reverse --version-sort)"

  if [[ "${out}" == "$(record_releases)" ]]; then
    echo "No update"
    return 0
  fi

  echo "${out}" >"${record}"
}

main
