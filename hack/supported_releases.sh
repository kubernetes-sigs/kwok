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

DIR="$(realpath "${DIR}")"

ROOT="$(realpath "${DIR}/..")"

record="${ROOT}/supported_releases.txt"

function latest_releases() {
  local ref_image=kindest/node
  token=$(curl "https://auth.docker.io/token?service=registry.docker.io&scope=repository:${ref_image}:pull" | jq -r '.token')
  tags=$(curl "https://registry-1.docker.io/v2/${ref_image}/tags/list" -H "Authorization: Bearer ${token}" | jq -r '.tags | .[]')
  echo "${tags}" | grep -e '^v\d\+\.\d\+\.\d\+$'
}

function record_releases() {
  cat "${record}"
}

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

function send_pr() {
  local status
  local branch
  status=$(git status -s)
  if [[ "${status}" == "" ]]; then
    echo "No modification"
    return 1
  fi
  if [[ "${status}" != " M supported_releases.txt" ]]; then
    echo "Modified unintended documents"
    echo "${status}"
    return 1
  fi

  if [[ "$(git config --global user.name)" == "" ]]; then
    git config --global user.name "bot"
  fi
  branch="bump-releases-$(date +"%Y%m%d")"
  git branch "${branch}"
  git checkout "${branch}"
  git add supported_releases.txt
  git commit -m "Bump supported_releases.txt"
  git push --set-upstream origin "${branch}"
  gh pr create --fill
}

function main() {
  out="$(bump $(record_releases) $(latest_releases) | sort --reverse --version-sort)"

  if [[ "${out}" == "${recored}" ]]; then
    return 0
  fi
  echo "${out}" >"${record}"
  send_pr
}

main
