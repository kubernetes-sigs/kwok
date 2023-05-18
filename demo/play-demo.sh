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

function br() {
  echo
}

function color() {
  local color="${1}"
  local text="${2}"
  echo -e "\033[1;${color}m${text}\033[0m"
}

PLAY_PWD="${PLAY_PWD:-"~/sigs.k8s.io/kwok"}"
PLAY_PS1="$(color 96 "${PLAY_PWD}")$(color 94 "$") "

function ps1() {
  local delay="${1}"
  echo -n "${PLAY_PS1}"
  if [[ "${delay}" != "" ]]; then
    sleep "${delay}"
  fi
}

# type_message prints a message to stdout at a human hand speed.
function type_message() {
  local message="$1"
  local delay="${2:-0.02}"
  local entry_delay="${3:-0.1}"
  for ((i = 0; i < ${#message}; i++)); do
    echo -n "${message:$i:1}"
    sleep "${delay}"
  done
  sleep "${entry_delay}"
  br
}

# type_and_exec_command prints a command to stdout and executes it.
function type_and_exec_command() {
  local command="$*"
  type_message "${command}" 0.01 0.5
  eval "${command}"
}

# play_file plays a file line by line.
function play_file() {
  local file="$1"
  while read -r line; do
    if [[ "${line}" =~ ^# ]]; then
      ps1 0.5
      type_message "${line}"
    elif [[ "${line}" == "" ]]; then
      ps1 2
      br
    else
      ps1 1
      type_and_exec_command "${line}"
    fi
  done <"${file}"
}

play_file "$1"
