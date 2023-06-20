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
export LC_ALL=C

# Copy from https://github.com/kubernetes-sigs/kind/blob/main/hack/release/get-contributors.sh

# query git for contributors since the tag
mapfile -t contributors < <(git log --format="%aN <%aE>" "${LAST_VERSION_TAG:?}.." | sort --unique)

# query github for usernames and output a bulleted list
contributor_logins=()
for contributor in "${contributors[@]}"; do
  # get a commit for this author
  commit_for_contributor="$(git log --author="${contributor}" --pretty=format:"%H" -1)"
  # lookup the  commit info to get the login
  contributor_login="$(
    curl \
      -sG \
      ${GH_TOKEN:+-H "Authorization: Bearer ${GH_TOKEN:?}"} \
      --data-urlencode "q=${contributor}" \
      "https://api.github.com/repos/${ORG}/${REPO}/commits/${commit_for_contributor}" |
      jq -r '.author | select(.login != null) | .login'
  )"
  if [[ "${contributor_login}" == "" ]]; then
    continue
  fi
  contributor_logins+=("${contributor_login}")
done

echo "Contributors since ${LAST_VERSION_TAG}:"

echo "${contributor_logins[@]}" | tr ' ' '\n' | sort --ignore-case --unique | awk '{print "- @"$0}'
