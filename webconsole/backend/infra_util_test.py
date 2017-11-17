# -*- coding: utf-8 -*-
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
# limitations under the License

"""Tests for infra_util.py"""

import unittest

from google.cloud import pubsub
from google.cloud import spanner
from mock import MagicMock
from mock import patch

from create_infra import constants
from create_infra import cloud_functions_builder
from create_infra import compute_builder
from create_infra import pubsub_builder
from create_infra import spanner_builder

from create_infra.resource_status import ResourceStatus
import infra_util


# pylint: disable=too-many-public-methods,protected-access,invalid-name
class InfraUtilTest(unittest.TestCase):
    """Unit tests for infra_util.py"""

    @staticmethod
    @patch.object(spanner, 'Client')
    @patch.object(spanner_builder, 'SpannerBuilder')
    @patch.object(pubsub, 'Client')
    @patch.object(pubsub_builder, 'PubSubBuilder')
    @patch.object(cloud_functions_builder, 'CloudFunctionsBuilder')
    @patch.object(compute_builder, 'ComputeBuilder')
    # pylint: disable=too-many-arguments
    def test_infrastructure_status_create_builders(
        compute_builder_mock, functions_builder_mock,
        pubsub_builder_mock, pubsub_client_mock,
        spanner_builder_mock, spanner_client_mock):
        """
        Tests infrastructure_status method constructing the right builders.
        """
        spanner_client_instance = spanner_client_mock.return_value
        pubsub_client_instance = pubsub_client_mock.return_value

        credentials = MagicMock()
        project_id = 'project'

        infra_util.infrastructure_status(credentials, project_id)

        spanner_builder_mock.assert_called_once_with(
            constants.SPANNER_INSTANCE, client=spanner_client_instance)
        spanner_client_mock.assert_called_once_with(credentials=credentials,
                                                    project=project_id)

        pubsub_builder_mock.assert_called_once_with(
            client=pubsub_client_instance)
        pubsub_client_mock.assert_called_once_with(credentials=credentials,
                                                   project=project_id)

        functions_builder_mock.assert_called_once_with(credentials=credentials,
                                                       project_id=project_id)

        compute_builder_mock.assert_called_once_with(credentials=credentials,
                                                     project_id=project_id)
    # pylint: enable=too-many-arguments

    def test_infrastructure_status_from_builders(self):
        """
        Tests _infrastructure_status_from_bldrs method.
        """
        def topic_and_subscriptions_status_side_effect(topic, subscriptions):
            # pylint: disable=missing-docstring, unused-argument
            return_value = ResourceStatus.UNKNOWN
            if topic == constants.LIST_TOPIC:
                return_value = ResourceStatus.RUNNING
            if topic == constants.LIST_PROGRESS_TOPIC:
                return_value = ResourceStatus.UNKNOWN
            if topic == constants.UPLOAD_GCS_TOPIC:
                return_value = ResourceStatus.DELETING
            if topic == constants.UPLOAD_GCS_PROGRESS_TOPIC:
                return_value = ResourceStatus.RUNNING
            return return_value

        spanner_bldr = MagicMock()
        pubsub_bldr = MagicMock()
        functions_bldr = MagicMock()
        compute_bldr = MagicMock()

        self.maxDiff = None
        spanner_bldr.database_status.return_value = ResourceStatus.NOT_FOUND
        pubsub_bldr.topic_and_subscriptions_status.side_effect = (
            topic_and_subscriptions_status_side_effect)
        functions_bldr.function_status.return_value = ResourceStatus.DEPLOYING
        compute_bldr.instance_status.return_value = ResourceStatus.UNKNOWN

        expected_status = {
            "cloudFunctionsStatus": ResourceStatus.DEPLOYING.name,
            "dcpStatus": ResourceStatus.UNKNOWN.name,
            "pubsubStatus": {
                "list": ResourceStatus.RUNNING.name,
                "listProgress": ResourceStatus.UNKNOWN.name,
                "uploadGCS": ResourceStatus.DELETING.name,
                "uploadGCSProgress": ResourceStatus.RUNNING.name
            },
            "spannerStatus": ResourceStatus.NOT_FOUND.name,
          }

        status = infra_util._infrastructure_status_from_bldrs(
            spanner_bldr, pubsub_bldr, functions_bldr, compute_bldr)
        self.assertDictEqual(status, expected_status)
