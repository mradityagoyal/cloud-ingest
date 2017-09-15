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

"""Default values for the cloud ingest infra-structure resource names.
"""

# Default values for cloud ingest infra-structure spanner instance.
SPANNER_INSTANCE = 'cloud-ingest-spanner-instance'
SPANNER_DATABASE = 'cloud-ingest-database'

# Default values for cloud ingest infra-structure pub/sub topics and
# subscriptions.
LIST_TOPIC = 'cloud-ingest-list'
LIST_SUBSCRIPTION = LIST_TOPIC
LIST_PROGRESS_TOPIC = 'cloud-ingest-list-progress'
LIST_PROGRESS_SUBSCRIPTION = LIST_PROGRESS_TOPIC

UPLOAD_GCS_TOPIC = 'cloud-ingest-copy'
UPLOAD_GCS_SUBSCRIPTION = UPLOAD_GCS_TOPIC
UPLOAD_GCS_PROGRESS_TOPIC = 'cloud-ingest-copy-progress'
UPLOAD_GCS_PROGRESS_SUBSCRIPTION = UPLOAD_GCS_PROGRESS_TOPIC

LOAD_BQ_TOPIC = 'cloud-ingest-loadbigquery'
LOAD_BQ_SUBSCRIPTION = LOAD_BQ_TOPIC
LOAD_BQ_PROGRESS_TOPIC = 'cloud-ingest-loadbigquery-progress'
LOAD_BQ_PROGRESS_SUBSCRIPTION = LOAD_BQ_PROGRESS_TOPIC

# Cloud ingest infra-structure default cloud function params.
LOAD_BQ_CLOUD_FN_NAME = 'cloud-ingest-gcs_to_bq_importer'
LOAD_BQ_CLOUD_FN_ENTRY_POINT = 'GcsToBq'
LOAD_BQ_CLOUD_FN_TIMEOUT_SECS = '540s'

# Cloud ingest infra-structure default DCP GCE instance params.
DCP_INSTANCE_NAME = 'cloud-ingest-dcp'
# TODO(b/63626194): Change with google official container registry.
DCP_INSTANCE_DOCKER_IMAGE = 'gcr.io/mbassiouny-test/cloud-ingest:dcp'
DCP_INSTANCE_CMD_LINE = '/cloud-ingest/dcpmain'
