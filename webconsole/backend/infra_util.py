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

"""Ingest infrastructure utility functions for webconsole backend."""

from google.cloud import pubsub
from google.cloud import spanner

from create_infra import constants
from create_infra import cloud_functions_builder
from create_infra import compute_builder
from create_infra import pubsub_builder
from create_infra import spanner_builder

# The cloud ingest pre-defined topics and subscriptions.
_TOPICS_SUBSCRIPTIONS = {
    'list': (constants.LIST_TOPIC,
             [constants.LIST_SUBSCRIPTION]),
    'listProgress': (constants.LIST_PROGRESS_TOPIC,
                     [constants.LIST_PROGRESS_SUBSCRIPTION]),
    'uploadGCS': (constants.UPLOAD_GCS_TOPIC,
                  [constants.UPLOAD_GCS_SUBSCRIPTION]),
    'uploadGCSProgress': (constants.UPLOAD_GCS_PROGRESS_TOPIC,
                          [constants.UPLOAD_GCS_PROGRESS_SUBSCRIPTION]),
    'loadBigQuery': (constants.LOAD_BQ_TOPIC,
                     [constants.LOAD_BQ_SUBSCRIPTION]),
    'loadBigQueryProgres': (constants.LOAD_BQ_PROGRESS_TOPIC,
                            [constants.LOAD_BQ_PROGRESS_SUBSCRIPTION]),
}

# pylint: disable=invalid-name
def _infrastructure_status_from_bldrs(spanner_bldr, pubsub_bldr,
                                      functions_bldr, compute_bldr):
    """Gets the ingest infrastructure deployment status. It uses the passed
    builder objects to check for resources statues.

    Args:
        spanner_bldr: SpannerBuilder to get the spanner database deployment
            status.
        pubsub_bldr: PubBuilder to get pub/sub topics and subscriptions
            deployment status.
        functions_bldr: CloudFunctionsBuilder to get the cloud functions
            deployment status.
        compute_bldr: ComputeBuilder to get the DCP GCE instance deployment
            status.

    Returns:
        A dictionary that contains all the infrastructure (spanner database, all
        pub/sub topics and subscriptions, cloud function and DCP GCE instance)
        deployment statuses. Each status is either a status string from one of
        the following values ('RUNNING', 'DEPLOYING', 'DELETING', 'FAILED',
        'NOT_FOUND', or 'UNKNOWN'), or a dictionary with that contains status
        strings. Sample return value:
        {
            'cloudFunctionsStatus': 'DEPLOYING',
            'dcpStatus': 'RUNNING',
            'pubsubStatus': {
                'list': 'RUNNING',
                'listProgress': 'RUNNING',
                'loadBigQuery': 'RUNNING',
                'loadBigQueryProgres': 'RUNNING',
                'uploadGCS': 'RUNNING',
                'uploadGCSProgress': 'RUNNING'
            },
            'spannerStatus': 'RUNNING'
        }
    """
    # TODO(b/65559194): Parallelize the requests to get infrastructure status.
    pubsub_status = {}
    for key, topic_subscriptions in _TOPICS_SUBSCRIPTIONS.iteritems():
        pubsub_status[key] = pubsub_bldr.topic_and_subscriptions_status(
            topic_subscriptions[0], topic_subscriptions[1]).name

    statuses = {
        'spannerStatus': spanner_bldr.database_status(
            constants.SPANNER_DATABASE).name,
        'pubsubStatus': pubsub_status,
        'dcpStatus': compute_bldr.instance_status(
            constants.DCP_INSTANCE_NAME).name,
        'cloudFunctionsStatus': functions_bldr.function_status(
            constants.LOAD_BQ_CLOUD_FN_NAME).name,
    }
    return statuses
# pylint: enable=invalid-name

def infrastructure_status(credentials, project_id):
    """Gets the ingest infrastructure status.

    Args:
        credentials: The credentials to use for querying the infrastructure.
        project_id: The project id.

    Returns:
        A dictionary contains all the infrastructure component statuses. Each
        status is a string from one of the following values ('RUNNING',
        'DEPLOYING', 'DELETING', 'FAILED', 'NOT_FOUND', or 'UNKNOWN').
    """
    spanner_client = spanner.Client(credentials=credentials, project=project_id)
    spanner_bldr = spanner_builder.SpannerBuilder(constants.SPANNER_INSTANCE,
                                                  client=spanner_client)

    pubsub_client = pubsub.Client(credentials=credentials, project=project_id)
    pubsub_bldr = pubsub_builder.PubSubBuilder(client=pubsub_client)

    functions_bldr = cloud_functions_builder.CloudFunctionsBuilder(
        credentials=credentials, project_id=project_id)

    compute_bldr = compute_builder.ComputeBuilder(credentials=credentials,
                                                  project_id=project_id)
    return _infrastructure_status_from_bldrs(
        spanner_bldr, pubsub_bldr, functions_bldr, compute_bldr)
