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

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

function blob_base() {
  local tag=${1}
  local repo
  repo="$(gh repo view --json url --jq '.url')"
  local url="${repo}/releases/download/${tag}"
  echo "${url}"
}

function update_index() {
  local index_dir="${1}"
  local tag=${2}
  local filepath="${3}"

  if ! gh release view "${tag}" >/dev/null; then
    echo "Please create release ${tag}"
    exit 1
  fi

  gh release upload "${tag}" "${filepath}"

  local url
  url="$(blob_base "${tag}")"

  helm repo index "${index_dir}" --merge "${index_dir}/index.yaml" --url "${url}"
}

function package_and_index() {
  local index_dir="${1}"
  local chart_dir="${2}"
  local chart_alias="${3}"

  local chart_name
  local chart_version
  local chart_app_version

  chart_name="$(yq eval '.name' "${chart_dir}/Chart.yaml")"
  chart_version="$(yq eval '.version' "${chart_dir}/Chart.yaml")"
  chart_app_version="$(yq eval '.appVersion' "${chart_dir}/Chart.yaml")"

  if yq -e eval ".entries.${chart_name}.[] | select(.version == \"${chart_version}\") | .version" "${index_dir}/index.yaml"; then
    echo "Version ${chart_version} already exists"
    return 0
  fi
  local guess_path="${index_dir}/${chart_name}-${chart_version}.tgz"
  if [[ -f "${guess_path}" ]]; then
    echo "File ${guess_path} already exists"
    return 0
  fi

  helm package "${chart_dir}" --destination "${index_dir}"

  if [[ "${chart_alias}" != "" ]]; then
    local tmp_file="${index_dir}/${chart_alias}-${chart_version}.tgz"
    mv "${guess_path}" "${tmp_file}"
    guess_path="${tmp_file}"
  fi

  update_index "${index_dir}" "${chart_app_version}" "${guess_path}"
}

chart_dir="./charts"
index_dir="${ROOT_DIR}/site/static/charts"

package_and_index "${index_dir}" "${chart_dir}/kwok" "kwok-chart" || :
package_and_index "${index_dir}" "${chart_dir}/operator" "kwok-operator-chart" || :
package_and_index "${index_dir}" "${chart_dir}/stage-fast" "kwok-stage-fast-chart" || :
package_and_index "${index_dir}" "${chart_dir}/metrics-usage" "kwok-metrics-usage-chart" || :
