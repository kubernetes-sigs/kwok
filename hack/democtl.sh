#!/usr/bin/env bash
# Copyright 2024 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

SIMULATE=false
COLS=86
ROWS=24
SVG_TERM=""
SVG_PROFILE=""

function color() {
  local color="${1}"
  local text="${2}"
  echo -e "\033[1;${color}m${text}\033[0m"
}

export PLAY_PS1="$ "

CACHE_DIR=${TMPDIR:-/tmp}/democtl
SELFPATH="$(realpath "$0")"
ARGS=()

export PATH="${CACHE_DIR}/py_modules/bin:${CACHE_DIR}/node_modules/.bin:${PATH}:${PATH}"

function usage() {
  echo "Usage: ${0} <input> <output> [--help] [options...]"
  echo "  <input> input file"
  echo "  <output> output file"
  echo "  --help show this help"
  echo "  --cols=${COLS} cols of the terminal"
  echo "  --rows=${ROWS} rows of the terminal"
  echo "  --ps1=${PLAY_PS1} ps1 of the recording"
  echo "  --term=${SVG_TERM} terminal type"
  echo "  --profile=${SVG_PROFILE} terminal profile"
}

# args parses the arguments.
function args() {
  local arg

  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --internal-simulate)
      SIMULATE="true"
      shift
      ;;
    --cols | --cols=*)
      [[ "${arg#*=}" != "${arg}" ]] && COLS="${arg#*=}" || { COLS="${2}" && shift; } || :
      shift
      ;;
    --rows | --rows=*)
      [[ "${arg#*=}" != "${arg}" ]] && ROWS="${arg#*=}" || { ROWS="${2}" && shift; } || :
      shift
      ;;
    --ps1 | --ps1=*)
      [[ "${arg#*=}" != "${arg}" ]] && PLAY_PS1="${arg#*=}" || { PLAY_PS1="${2}" && shift; } || :
      shift
      ;;
    --term | --term=*)
      [[ "${arg#*=}" != "${arg}" ]] && SVG_TERM="${arg#*=}" || { SVG_TERM="${2}" && shift; } || :
      shift
      ;;
    --profile | --profile=*)
      [[ "${arg#*=}" != "${arg}" ]] && SVG_PROFILE="${arg#*=}" || { SVG_PROFILE="${2}" && shift; } || :
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    --*)
      echo "Unknown argument: ${arg}"
      usage
      exit 1
      ;;
    *)
      ARGS+=("${arg}")
      shift
      ;;
    esac
  done
}

# command_exist checks if the command exists.
function command_exist() {
  local command="${1}"
  type "${command}" >/dev/null 2>&1
}

# install_asciinema installs asciinema.
function install_asciinema() {
  if command_exist asciinema; then
    return 0
  elif command_exist pip3; then
    pip3 install asciinema --target "${CACHE_DIR}/py_modules" >&2
  else
    echo "asciinema is not installed" >&2
    return 1
  fi
}

# install_svg_term_cli installs svg-term-cli.
function install_svg_term_cli() {
  if command_exist svg-term; then
    return 0
  elif command_exist npm; then
    npm install --save-dev svg-term-cli --prefix "${CACHE_DIR}" >&2
  else
    echo "svg-term is not installed" >&2
    return 1
  fi
}

# install_svg_to_video installs svg-to-video.
function install_svg_to_video() {
  if command_exist svg-to-video; then
    return 0
  elif command_exist npm; then
    npm install --save-dev https://github.com/wzshiming/svg-to-video --prefix "${CACHE_DIR}" >&2
  else
    echo "svg-to-video is not installed" >&2
    return 1
  fi
}

# ext_file returns the extension of the input file.
function ext_file() {
  local file="${1}"
  echo "${file##*.}"
}

# ext_replace replaces the extension of the input file with the output extension.
function ext_replace() {
  local file="${1}"
  local ext="${2}"
  echo "${file%.*}.${ext}"
}

# demo2cast converts the input demo file to the output cast file.
function demo2cast() {
  local input="${1}"
  local output="${2}"
  echo "Recording ${input} to ${output}" >&2

  asciinema rec \
    "${output}" \
    --overwrite \
    --cols "${COLS}" \
    --rows "${ROWS}" \
    --env "" \
    --command "bash ${SELFPATH} ${input} --internal-simulate --ps1='${PLAY_PS1}'"
}

# cast2svg converts the input cast file to the output svg file.
function cast2svg() {
  local input="${1}"
  local output="${2}"
  local args=()
  echo "Converting ${input} to ${output}" >&2

  if [[ "${SVG_TERM}" != "" ]]; then
    args+=("--term" "${SVG_TERM}")
  fi

  if [[ "${SVG_PROFILE}" != "" ]]; then
    args+=("--profile" "${SVG_PROFILE}")
  fi
  svg-term \
    --in "${input}" \
    --out "${output}" \
    --window \
    "${args[@]}"
}

# svg2video converts the input svg file to the output video file.
function svg2video() {
  local input="${1}"
  local output="${2}"
  echo "Converting ${input} to ${output}" >&2

  svg-to-video \
    "${input}" \
    "${output}" \
    --delay-start 5 \
    --headless
}

# convert converts the input file to the output file.
# The input file can be a demo, cast, or svg file.
# The output file can be a cast, svg, or mp4 file.
function convert() {
  local input="${1}"
  local output="${2}"

  local castfile
  local viedofile

  local outext
  local inext

  outext=$(ext_file "${output}")
  inext=$(ext_file "${input}")
  case "${outext}" in
  cast)
    case "${inext}" in
    demo)
      install_asciinema

      demo2cast "${input}" "${output}"
      return 0
      ;;
    *)
      echo "Unsupported input file type: ${inext}"
      return 1
      ;;
    esac
    ;;
  svg)
    case "${inext}" in
    cast)
      install_svg_term_cli

      cast2svg "${input}" "${output}"
      return 0
      ;;
    demo)
      install_asciinema
      install_svg_term_cli

      castfile=$(ext_replace "${output}" "cast")
      demo2cast "${input}" "${castfile}"
      cast2svg "${castfile}" "${output}"
      return 0
      ;;
    *)
      echo "Unsupported input file type: ${inext}"
      return 1
      ;;
    esac
    ;;
  mp4)
    case "${inext}" in
    svg)
      install_svg_to_video

      svg2video "${input}" "${output}"
      return 0
      ;;
    cast)
      install_svg_term_cli
      install_svg_to_video

      viedofile=$(ext_replace "${output}" "svg")
      cast2svg "${input}" "${viedofile}"
      svg2video "${viedofile}" "${output}"
      return 0
      ;;
    demo)
      install_asciinema
      install_svg_term_cli
      install_svg_to_video

      viedofile=$(ext_replace "${output}" "svg")
      castfile=$(ext_replace "${output}" "cast")
      demo2cast "${input}" "${castfile}"
      cast2svg "${castfile}" "${viedofile}"
      svg2video "${viedofile}" "${output}"
      return 0
      ;;
    *)
      echo "Unsupported input file type: ${inext}"
      return 1
      ;;
    esac
    ;;
  *)
    echo "Unsupported output file type: ${outext}"
    return 1
    ;;
  esac
}

# br prints a new line.
# this function is used to simulate typing.
function br() {
  echo
}

# ps1 prints the ps1 with a delay.
# this function is used to simulate typing.
function ps1() {
  local delay="${1}"
  echo -e -n "${PLAY_PS1}"
  if [[ "${delay}" != "" ]]; then
    sleep "${delay}"
  fi
}

# type_message prints a message to stdout at a human hand speed.
# this function is used to simulate typing.
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
# this function is used to simulate typing.
function type_and_exec_command() {
  local command="$*"
  type_message "${command}" 0.01 0.5
  eval "${command}"
}

# play_file plays a file line by line.
# this function is used to simulate typing.
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

function main() {
  if [[ "${#ARGS[*]}" -lt 1 ]]; then
    usage
    exit 1
  fi

  INPUT_FILE="${ARGS[0]}"

  # Only for asciinema recording command.
  if [[ "${SIMULATE}" == "true" ]]; then
    play_file "${INPUT_FILE}"
    exit 0
  fi

  if [[ "${#ARGS[*]}" -gt 1 ]]; then
    OUTPUT_FILE="${ARGS[1]}"
  else
    # If the output file is not specified, use the same name as the input file.
    OUTPUT_FILE="$(ext_replace "${INPUT_FILE}" "svg")"
  fi

  convert "${INPUT_FILE}" "${OUTPUT_FILE}"
}

args "$@"
main
