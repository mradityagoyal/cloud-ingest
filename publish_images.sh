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

#!/bin/bash

# Exit the script on the first failure.
set -e

# Capture this script directory.
SCRIPT_DIR="$( dirname "${BASH_SOURCE[0]}" )"

# Move to this script dir.
pushd "$SCRIPT_DIR"

# Return back to the original dir.
trap popd EXIT

# TODO(b/63626194): Change with google official container registery.
PROJECT_ID="mbassiouny-test"

fail() {
  local _red="\\033[1;31m"
  local _normal="\\033[0;39m"
  [ -n "$*" ] && >&2 printf "${_red}$*${_normal}\n"
  exit 1
}

if [ $# -ne 1 ] || ( [ "$1" != "base" ] && [ "$1" != "dcp" ] ); then
 fail "Should provide 1 argument (base|dcp)"
fi

if [ "$1" = "base" ]; then
  docker build -t cloud-ingest:base -f Dockerfile-base .
  docker tag cloud-ingest:base "gcr.io/$PROJECT_ID/cloud-ingest:base"
  gcloud docker -- push "gcr.io/$PROJECT_ID/cloud-ingest:base"
else
  # Build the dcp go binary
  go build ./dcp/dcpmain
  docker build -t cloud-ingest:dcp -f Dockerfile-dcp .
  # Clean up the dcp binary after building the image.
  rm -f dcpmain
  docker tag cloud-ingest:dcp "gcr.io/$PROJECT_ID/cloud-ingest:dcp"
  gcloud docker -- push "gcr.io/$PROJECT_ID/cloud-ingest:dcp"
fi
