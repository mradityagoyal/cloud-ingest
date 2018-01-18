# Copyright 2018 Google Inc. All Rights Reserved.
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

# Script to build and/or tear down the cloud ingest infra structure.

#!/bin/bash

source $(dirname $0)/common.inc

SPANNER_INSTANCE="cloud-ingest-spanner-instance"
SPANNER_DATABASE="cloud-ingest-database"

PROCESS_LIST_TOPIC="cloud-ingest-process-list"
PROCESS_LIST_SUBSCRIPTION="$PROCESS_LIST_TOPIC"

DCP_SERVICE_ACCOUNT="cloud-ingest-dcp"

DCP_CLUSTER_NAME="cloud-ingest-dcp-cluster"

usage() {
  echo "Usage: $(basename $BASH_SOURCE) (--projectid|-p) <project_id>"\
    "[(--container-image|-i) <dcp_container_image>]" \
    "[(--create|-c)] [(--teardown|-t)] [(--help|-h)]"
  [ -n "$*" ] && fail "$*"
}

parse_params() {
  while [[ $# -gt 0 ]]
  do
  key="$1"

  case $key in
      -p|--projectid)
      PROJECT_ID="$2"
      shift # past argument
      shift # past value
      ;;
      -i|--container-image)
      DCP_CONTAINER_IMAGE="$2"
      shift # past argument
      shift # past value
      ;;
      -c|--create)
      CREATE=true
      shift # past argument
      ;;
      -t|--teardown)
      TEAR_DOWN=true
      shift # past argument
      ;;
      -sdcp|--skip-dcp)
      SKIP_DCP=true
      shift # past argument
      ;;
      -h|--help)
      usage
      exit 0
      ;;
      *)    # unknown option
      fail "Unknown argument $key."
      ;;
  esac
  done
}

parse_params $*

[ -z "$PROJECT_ID" ] && \
  usage "project id should be set."

DCP_SERVICE_ACCOUNT_EMAIL="$DCP_SERVICE_ACCOUNT@$PROJECT_ID.iam.gserviceaccount.com"

if [ "$TEAR_DOWN" = true ] ; then
  read -p "Are you sure you want to tear down the infrastructure? (y/n) " -r
  if ! [[ $REPLY =~ ^[Yy]$ ]]
  then
      fail "Skip tearing down!"
  fi

  echo "Tearing down DCP cluster."
  gcloud container clusters delete "$DCP_CLUSTER_NAME" \
    --quiet --project="$PROJECT_ID"

  echo "Removing DCP service account."
  gcloud iam service-accounts delete "$DCP_SERVICE_ACCOUNT_EMAIL" \
    --quiet --project="$PROJECT_ID"

  echo "Tearing down Spanner."
  gcloud spanner instances delete "$SPANNER_INSTANCE" \
    --quiet --project="$PROJECT_ID"

  echo "Tearing down Pub/Sub."
  gcloud pubsub subscriptions delete "$PROCESS_LIST_SUBSCRIPTION" \
    --project="$PROJECT_ID"
  gcloud pubsub topics delete "$PROCESS_LIST_TOPIC" --project="$PROJECT_ID"
fi

if [ "$CREATE" = true ] ; then
  # Exit the script on the first failure.
  set -e

  [ -z "$DCP_CONTAINER_IMAGE" ] && \
    usage "container image should be set when creating infrastructure."

  echo "Creating Spanner instance."
  gcloud spanner instances create "$SPANNER_INSTANCE" \
    --config=regional-us-central1 \
    --description="Cloud Ingest Spanner Instance" --nodes=1 \
    --project="$PROJECT_ID"

  echo "Creating Spanner database."
  gcloud spanner databases create "$SPANNER_DATABASE" \
    --project="$PROJECT_ID" \
    --instance="$SPANNER_INSTANCE" \
    --ddl="$(sed ':a;N;$!ba;s/\n\n/;\n\n/g' webconsole/backend/create_infra/schema.ddl)"

  echo "Creating Pub/Sub."
  gcloud pubsub topics create "$PROCESS_LIST_TOPIC" --project="$PROJECT_ID"
  gcloud pubsub subscriptions create "$PROCESS_LIST_SUBSCRIPTION" \
    --topic "$PROCESS_LIST_TOPIC" --ack-deadline=30 --project="$PROJECT_ID"

  echo "Creating the DCP service account."
  gcloud iam service-accounts create "$DCP_SERVICE_ACCOUNT" \
    --display-name "Cloud Ingest DCP service account"

  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$DCP_SERVICE_ACCOUNT_EMAIL" \
    --role=roles/editor --no-user-output-enabled

  if [ "$SKIP_DCP" = true ] ; then
    echo "Skip creating the DCP K8 cluster."
  else
    echo "Creating DCP K8 cluster."
    gcloud container clusters create "$DCP_CLUSTER_NAME" \
      --project="$PROJECT_ID" --service-account="$DCP_SERVICE_ACCOUNT_EMAIL"

    echo "Deploying the DCP container into the cluster."
    kubectl run dcp --image="$DCP_CONTAINER_IMAGE" --replicas=1 \
      --command -- ./dcpmain -projectid="$PROJECT_ID" -logtostderr
  fi
fi
exit 0
