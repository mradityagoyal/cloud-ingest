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

from google.cloud import spanner
from mock import MagicMock
from mock import patch

from create_infra import constants
from create_infra import compute_builder
from create_infra import pubsub_builder
from create_infra import spanner_builder
from proto.tasks_pb2 import ResourceStatus

import infra_util


# pylint: disable=too-many-public-methods,protected-access,invalid-name
class InfraUtilTest(unittest.TestCase):
    """Unit tests for infra_util.py"""

    @staticmethod
    @patch.object(pubsub_builder, 'PubSubBuilder')
    # pylint: disable=too-many-arguments
    def test_infrastructure_status_create_builders(pubsub_builder_mock):
        """
        Tests infrastructure_status method constructing the right builders.
        """
        credentials = MagicMock()
        project_id = 'project'

        infra_util.infrastructure_status(credentials, project_id)

        pubsub_builder_mock.assert_called_once_with(credentials=credentials,
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
            if topic == constants.COPY_TOPIC:
                return_value = ResourceStatus.DELETING
            if topic == constants.COPY_PROGRESS_TOPIC:
                return_value = ResourceStatus.RUNNING
            return return_value

        pubsub_bldr = MagicMock()

        self.maxDiff = None
        pubsub_bldr.topic_and_subscriptions_status.side_effect = (
            topic_and_subscriptions_status_side_effect)

        expected_status = {
            "pubsubStatus": {
                "list": ResourceStatus.RUNNING,
                "listProgress": ResourceStatus.UNKNOWN,
                "copy": ResourceStatus.DELETING,
                "copyProgress": ResourceStatus.RUNNING
            },
          }

        status = infra_util._infrastructure_status_from_bldrs(pubsub_bldr)
        self.assertDictEqual(status, expected_status)
