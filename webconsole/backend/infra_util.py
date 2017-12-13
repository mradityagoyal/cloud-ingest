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

from google.cloud import spanner
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import PreconditionFailed
from os import path

from create_infra import constants
from create_infra import compute_builder
from create_infra import pubsub_builder
from create_infra import spanner_builder
from util import dict_has_values_recursively
from util import dict_values_are_recursively

from proto.tasks_pb2 import ResourceStatus

_CURRENT_DIR = path.dirname(path.realpath(__file__))

# The cloud ingest pre-defined topics and subscriptions.
_TOPICS_SUBSCRIPTIONS = {
    'list': (constants.LIST_TOPIC,
             [constants.LIST_SUBSCRIPTION], 30),
    'listProgress': (constants.LIST_PROGRESS_TOPIC,
                     [constants.LIST_PROGRESS_SUBSCRIPTION], 30),
    'processList': (constants.PROCESS_LIST_TOPIC,
                    [constants.PROCESS_LIST_SUBSCRIPTION], 30),
    'uploadGCS': (constants.UPLOAD_GCS_TOPIC,
                  [constants.UPLOAD_GCS_SUBSCRIPTION], 30),
    'uploadGCSProgress': (constants.UPLOAD_GCS_PROGRESS_TOPIC,
                          [constants.UPLOAD_GCS_PROGRESS_SUBSCRIPTION], 30),
}

# pylint: disable=invalid-name
def _infrastructure_status_from_bldrs(spanner_bldr, pubsub_bldr, compute_bldr):
    """Gets the ingest infrastructure deployment status. It uses the passed
    builder objects to check for resources statues.

    Args:
        spanner_bldr: SpannerBuilder to get the spanner database deployment
            status.
        pubsub_bldr: PubBuilder to get pub/sub topics and subscriptions
            deployment status.
        compute_bldr: ComputeBuilder to get the DCP GCE instance deployment
            status.

    Returns:
        A dictionary that contains all the infrastructure (spanner database, all
        pub/sub topics and subscriptions, and DCP GCE instance)
        deployment statuses. Each status is either a status string from one of
        the following values ('RUNNING', 'DEPLOYING', 'DELETING', 'FAILED',
        'NOT_FOUND', or 'UNKNOWN'), or a dictionary with that contains status
        strings. Sample return value:
        {
            'dcpStatus': 'RUNNING',
            'pubsubStatus': {
                'list': 'RUNNING',
                'listProgress': 'RUNNING',
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
            topic_subscriptions[0], topic_subscriptions[1])

    statuses = {
        'spannerStatus': spanner_bldr.database_status(
            constants.SPANNER_DATABASE),
        'pubsubStatus': pubsub_status,
        'dcpStatus': compute_bldr.instance_status(
            constants.DCP_INSTANCE_NAME),
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

    pubsub_bldr = pubsub_builder.PubSubBuilder(
        credentials=credentials, project_id=project_id)

    compute_bldr = compute_builder.ComputeBuilder(credentials=credentials,
                                                  project_id=project_id)
    return _infrastructure_status_from_bldrs(
        spanner_bldr, pubsub_bldr, compute_bldr)

def create_infrastructure(credentials, project_id, dcp_docker_image=None):
    """Creates the ingest infrastructure. Makes sure that all the infrastructure
    components does not exist before the creation.

    Args:
        credentials: The credentials to use for creating the infrastructure.
        project_id: The project id.
        dcp_docker_image: The dcp docker image to use. The DCP GCE istance won't
            created if this value is None.

    Raises:
        Conflict: If any of the infrastructure components exists.
    """
    # Creating the builders.
    spanner_client = spanner.Client(credentials=credentials, project=project_id)
    spanner_bldr = spanner_builder.SpannerBuilder(constants.SPANNER_INSTANCE,
                                                  client=spanner_client)

    pubsub_bldr = pubsub_builder.PubSubBuilder(
        credentials=credentials, project_id=project_id)

    compute_bldr = compute_builder.ComputeBuilder(credentials=credentials,
                                                  project_id=project_id)

    # Checking the infrastructure deployment status before creating it.
    infra_statuses = _infrastructure_status_from_bldrs(
        spanner_bldr, pubsub_bldr, compute_bldr)
    # Make sure all infrastructure components are not found.
    if not dict_values_are_recursively(infra_statuses,
                                       ResourceStatus.NOT_FOUND):
        raise Conflict('All the infrastructure resource (Spanner, Pub/Sub, and '
                       'DCP GCE instance) should not '
                       'exists before creating an infrastructure')

    # Create the spanner instance/database.
    spanner_bldr.create_instance()
    spanner_bldr.create_database(
        constants.SPANNER_DATABASE,
        path.join(_CURRENT_DIR, 'create_infra/schema.ddl'))

    # Create the topics and subscriptions.
    for topic_subscriptions in _TOPICS_SUBSCRIPTIONS.itervalues():
        pubsub_bldr.create_topic_and_subscriptions(
            topic_subscriptions[0], topic_subscriptions[1],
            ack_deadline=topic_subscriptions[2])

    # Create the DCP GCE instance.
    # TODO(b/65753224): Support of not creating the DCP GCE as part of the
    # create infrastructure. This will enable easily creation of dev
    # environments.
    if dcp_docker_image:
        compute_bldr.create_instance_async(
            constants.DCP_INSTANCE_NAME, dcp_docker_image,
            constants.DCP_INSTANCE_CMD_LINE, ["-projectid="+project_id])

def tear_infrastructure(credentials, project_id):
    """Tears the ingest infrastructure. Makes sure that all the infrastructure
    components are not deploying or deleting before tearing down.

    Args:
        credentials: The credentials to use for tearing the infrastructure.
        project_id: The project id.

    Raises:
        PreconditionFailed: If any of the infrastructure components is deploying
            or deleting.
    """
    # Creating the builders.
    spanner_client = spanner.Client(credentials=credentials, project=project_id)
    spanner_bldr = spanner_builder.SpannerBuilder(constants.SPANNER_INSTANCE,
                                                  client=spanner_client)

    pubsub_bldr = pubsub_builder.PubSubBuilder(
        credentials=credentials, project_id=project_id)

    compute_bldr = compute_builder.ComputeBuilder(credentials=credentials,
                                                  project_id=project_id)

    # Checking the infrastructure deployment status before creating it.
    infra_statuses = _infrastructure_status_from_bldrs(
        spanner_bldr, pubsub_bldr, compute_bldr)
    # Make sure all infrastructure components are not deploying or deleting.
    if dict_has_values_recursively(infra_statuses,
                                   set([ResourceStatus.DEPLOYING,
                                        ResourceStatus.DELETING])):
        raise PreconditionFailed(
            'All the infrastructure resources (Spanner, Pub/Sub, and DCP GCE '
            'instance) should not be deploying or '
            'deleting when tearing down infrastructure.')

    # Delete the spanner instance.
    spanner_bldr.delete_instance()

    # Delete the topics and subscriptions.
    for topic_subscriptions in _TOPICS_SUBSCRIPTIONS.itervalues():
        pubsub_bldr.delete_topic_and_subscriptions(topic_subscriptions[0])

    # Delete the DCP GCE instance.
    compute_bldr.delete_instance_async(constants.DCP_INSTANCE_NAME)
