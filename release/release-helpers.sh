#!/bin/bash

# GIT Commit ID (SHA1 Hash) regex
GIT_COMMIT_REGEX=[0-9a-f]{40}$

# GCS_STAGING_PREFIX is the GCS location for builds
GCS_STAGING_PREFIX=gs://cloud-ingest-rapid/agent

# GCS_VERSIONS_PREFIX is the GCS 'folder' in which we archive all builds of the
# agent, by version.
GCS_VERSIONS_PREFIX=gs://cloud-ingest/agent/versions

# VERSION_REGEX is a regular expression that all our agent version numbers must
# match.
VERSION_REGEX=^v[0-9]+\.[0-9]+\.[0-9]+$

# GCS_CANARY_PREFIX is the GCS 'folder' in which we archive all test
# builds of the agents which are used in the auto-update integration tests.
GCS_CANARY_PREFIX=gs://cloud-ingest-canary-candidate/agent

# die outputs a message and exits with a non-zero status.
function die() {
  echo -e >&2 "$@"
  exit 1
}