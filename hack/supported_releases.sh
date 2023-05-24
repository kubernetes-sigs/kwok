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

ROOT_DIR="$(realpath "${DIR}/..")"

record="${ROOT_DIR}/supported_releases.txt"

# For compatibility with the Kind runtime, get the labels from Kind's node images
function latest_releases() {
  local ref_image=kindest/node
  local auth_data
  local token
  local list_data
  local tags
  auth_data="$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:${ref_image}:pull")"
  token="$(echo "${auth_data}" | jq -r '.token')"
  if [[ "${token}" == "" ]]; then
    echo "Failed get token" >&2
    echo "${auth_data}" >&2
    return 1
  fi

  list_data="$(curl -s "https://registry-1.docker.io/v2/${ref_image}/tags/list" -H "Authorization: Bearer ${token}")"
  tags="$(echo "${list_data}" | jq -r '.tags | .[]' | grep -e '^v[0-9]\+\.[0-9]\+\.[0-9]\+$' | sed 's/v//g')"
  if [[ "${tags}" == "" ]]; then
    echo "Failed get list" >&2
    echo "${list_data}" >&2
    return 1
  fi
  echo "${tags}"
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

function send_pr() {
  local status
  local branch
  local body
  local diff_data

  status=$(git status -s)
  if [[ "${status}" == "" ]]; then
    echo "No modification"
    return 1
  fi

  # Use the fixed branch as the key to prevent duplicate PRs from being created
  branch="dependabot/bump-releases"

  if [[ "$(git branch -r --list "origin/${branch}")" == *"origin/${branch}"* ]]; then
    echo "Remote branch already exists"
    return 0
  fi
  if ! git checkout -B "${branch}"; then
    echo "Branch already exists"
    return 0
  fi

  if [[ "$(git config --global user.name)" == "" ]]; then
    git config --global user.name "github-actions"
    git config --global user.email "actions@github.com"
  fi

  diff_data="$(git diff)"

  git commit -a -m "Bump supported_releases.txt"
  if ! git push --set-upstream origin "${branch}"; then
    echo "Failed push branch ${branch}"
    return 1
  fi

  body="
\`\`\` diff

${diff_data}

\`\`\`

/kind feature
"
  gh pr create --title="Automated Bump supported_releases.txt" --body="${body}"
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

  # Update feature gate data
  "${ROOT_DIR}/pkg/kwokctl/k8s/feature_gates_data.sh" "$(echo "${out}" | head -n 1 | awk -F. '{print $2}')"

  # Update the generated files
  make -C "${ROOT_DIR}" update

  if [[ "${SEND_PR}" == "true" ]]; then
    send_pr
  fi
}

main
