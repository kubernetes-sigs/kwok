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

# get architecture of the machine
function arch() {
  local arch
  arch="$(uname -m)"

  case "${arch}" in
  x86_64 | amd64 | x64)
    echo "amd64"
    ;;
  i386 | i486 | i586 | i686 | x86)
    echo "386"
    ;;
  armv8* | aarch64* | arm64)
    echo "arm64"
    ;;
  armv7* | armhf)
    echo "arm"
    ;;
  *)
    echo "${arch}"
    ;;
  esac
}

# get os of the system
function os() {
  local os
  os="$(uname -s)"

  echo "${os}" | tr '[:upper:]' '[:lower:]'
}

echo "$(os)/$(arch)"
