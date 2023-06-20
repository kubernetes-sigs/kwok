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

DIR="$(dirname "${BASH_SOURCE[0]}")"

function change_notes() {
  "${DIR}"/get-change-notes.sh
  echo
}

function contributors() {
  echo "## Contributors "
  echo
  echo "Thank you to everyone who contributed to this release! ❤️"
  echo
  echo "Users whose commits are in this release (alphabetically by user name)"
  echo
  "${DIR}"/get-contributors.sh
  echo
  echo "And thank you very much to everyone else not listed here who contributed in other ways like filing issues, giving feedback, etc. 🙏"
}

output="${DIR}"/CHANGELOG.md
echo >"${output}"
change_notes >>"${output}"
contributors >>"${output}"
