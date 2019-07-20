#!/bin/bash

# GCS_VERSIONS_PREFIX is the GCS 'folder' in which we archive all builds of the
# agent, by version.
GCS_VERSIONS_PREFIX=gs://cloud-ingest/agent/versions

# VERSION_REGEX is a regular expression that all our agent version numbers must
# match.
VERSION_REGEX=^v[0-9]+\.[0-9]+\.[0-9]+$

# GCS_CANARY_PREFIX is the GCS 'folder' in which we archive all test
# builds of the agents which are used in the auto-update integration tests.
GCS_CANARY_PREFIX=gs://cloud-ingest-canary/agent

# die outputs a message and exits with a non-zero status.
function die() {
  echo -e >&2 "$@"
  exit 1
}

# prompt_with_regex keeps asking for input until it matches the provided regex.
function prompt_with_regex() {
  local msg=$1
  local regex=$2
  local input
  while true; do
    read -p "$msg: " input
    if [[ $input =~ $regex ]]; then # matches regex
      echo $input
      break
    fi
  done
}
