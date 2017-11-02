# Copyright 2017 Google Inc. All Rights Reserved.
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

# The script builds and publishes a docker images based on the passed param.
# This param should be one of:
#   base: Publish the cloud ingest base image which contains the cloud ingest
#         dependency packages pre-installed.
#   dcp: Publish the stable cloud ingest data control plane image.
#   test: Publish the cloud ingest data control plane image used for testing.
#   dev: Publish the local dev cloud ingest data control plane image used for
#        testing local changes.

#!/bin/bash

# Exit the script on the first failure.
set -e

# Capture this script directory.
SCRIPT_DIR="$( dirname "${BASH_SOURCE[0]}" )"

# Move to this script dir.
pushd "$SCRIPT_DIR"

# Return back to the original dir.
trap popd EXIT

# TODO(b/63626194): Change with google official container registry.
PROJECT_ID="mbassiouny-test"

fail() {
  local _red="\\033[1;31m"
  local _normal="\\033[0;39m"
  [ -n "$*" ] && >&2 printf "${_red}$*${_normal}\n"
  exit 1
}

if [ $# -ne 1 ] || ( [ "$1" != "base" ] && [ "$1" != "dcp" ] && \
                     [ "$1" != "test" ] && [ "$1" != "perf" ] && \
                     [ "$1" != "dev" ] ); then
 fail "Should provide 1 argument (base|dcp|test|perf|dev)"
fi

if [ "$1" = "base" ]; then
  docker build -t cloud-ingest:base -f Dockerfile-base .
  docker tag cloud-ingest:base "gcr.io/$PROJECT_ID/cloud-ingest:base"
  gcloud docker -- push "gcr.io/$PROJECT_ID/cloud-ingest:base"
else
  # Get the latest base image.
  gcloud docker -- pull "gcr.io/$PROJECT_ID/cloud-ingest:base"
  if [ "$1" = "dev" ]; then
    label="$USER"
  else
    label="$1"
  fi
  # Build the dcp go binary
  go build ./dcp/dcpmain
  docker build -t "cloud-ingest:$label" -f Dockerfile-dcp .
  # Clean up the dcp binary after building the image.
  rm -f dcpmain
  docker tag "cloud-ingest:$label" "gcr.io/$PROJECT_ID/cloud-ingest:$label"
  gcloud docker -- push "gcr.io/$PROJECT_ID/cloud-ingest:$label"
fi
