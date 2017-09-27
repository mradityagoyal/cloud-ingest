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

"""Tests for pubsub_builder.py"""

import unittest

# pylint: disable=relative-import
from google.cloud import pubsub
from google.gax import GaxError
from grpc import StatusCode
from mock import MagicMock
from mock import call
from mock import patch

from pubsub_builder import PubSubBuilder
from test.testutils import get_rpc_error_with_status_code


# pylint: disable=too-many-public-methods
class PubsubBuilderTest(unittest.TestCase):
    """Unit tests for pubsub_builder.py"""

    def setUp(self):
        def topic_path_mock_return(project_id, topic_name):
            # pylint: disable=missing-docstring, unused-argument
            return 'projects/%s/topics/%s' % (project_id, topic_name)

        def subscription_path_mock_return(project_id, topic_name):
            # pylint: disable=missing-docstring, unused-argument
            return 'projects/%s/subscriptions/%s' % (project_id, topic_name)

        self.pub_client_mock = MagicMock()
        self.pub_client_mock.topic_path.side_effect = topic_path_mock_return
        self.pub_patcher = patch.object(pubsub, 'PublisherClient',
                                        return_value=self.pub_client_mock)
        self.pub_patcher.start()

        self.sub_client_mock = MagicMock()
        self.sub_client_mock.topic_path.side_effect = topic_path_mock_return
        self.sub_client_mock.subscription_path.side_effect = (
            subscription_path_mock_return)
        self.sub_patcher = patch.object(pubsub, 'SubscriberClient',
                                        return_value=self.sub_client_mock)
        self.sub_patcher.start()

        self.builder = PubSubBuilder(project_id='myproject')

    def tearDown(self):
        self.pub_patcher.stop()
        self.sub_patcher.stop()

    def test_create_topic(self):
        """Tests creating a pubsub topic."""
        self.builder.create_topic('mytopic')
        self.pub_client_mock.create_topic.assert_called_once_with(
            'projects/myproject/topics/mytopic')

    def test_create_subscription(self):
        """Tests creating a topic subscription."""
        self.builder.create_subscription('mytopic', 'mysub', ack_deadline=30)

        self.sub_client_mock.create_subscription.assert_called_once_with(
            'projects/myproject/subscriptions/mysub',
            'projects/myproject/topics/mytopic', ack_deadline_seconds=30)

    def test_create_topic_subs(self):
        """Tests creating a pubsub topic and associated subscriptions."""
        self.builder.create_topic_and_subscriptions('mytopic', ['sub1', 'sub2'])

        self.pub_client_mock.create_topic.assert_called_once_with(
            'projects/myproject/topics/mytopic')

        expected_calls = [
            call('projects/myproject/subscriptions/sub1',
                 'projects/myproject/topics/mytopic', ack_deadline_seconds=15),
            call('projects/myproject/subscriptions/sub2',
                 'projects/myproject/topics/mytopic', ack_deadline_seconds=15)
        ]
        self.assertItemsEqual(
            expected_calls,
            self.sub_client_mock.create_subscription.mock_calls)

    def test_delete_raises_gax(self):
        """Tests deleting raises unknown GAX error."""
        self.pub_client_mock.list_topic_subscriptions.side_effect = GaxError(
            "msg", get_rpc_error_with_status_code(StatusCode.UNAUTHENTICATED))

        self.assertRaises(
            GaxError, self.builder.delete_topic_and_subscriptions, 'mytopic')

    def test_delete_non_exists_topic(self):
        """Tests deleting a non-exists pubsub topic."""
        self.pub_client_mock.list_topic_subscriptions.side_effect = GaxError(
            "msg", get_rpc_error_with_status_code(StatusCode.NOT_FOUND))

        self.builder.delete_topic_and_subscriptions('mytopic')
        self.pub_client_mock.delete_topic.assert_not_called()
        self.sub_client_mock.delete_subscription.assert_not_called()

    def test_delete_topic_subs(self):
        """Tests deleting a pubsub topic with all of its subscriptions."""
        topic_path = 'projects/myproject/topics/mytopic'
        sub1_path = 'projects/myproject/subscriptions/sub1'
        sub2_path = 'projects/myproject/subscriptions/sub2'

        self.pub_client_mock.list_topic_subscriptions.return_value = [
            sub1_path,
            sub2_path,
        ]
        self.builder.delete_topic_and_subscriptions('mytopic')
        self.pub_client_mock.list_topic_subscriptions.assert_called_once_with(
            topic_path)
        self.pub_client_mock.delete_topic.assert_called_once_with(topic_path)

        expected_calls = [
            call(sub1_path),
            call(sub2_path),
        ]

        self.assertItemsEqual(
            expected_calls, self.sub_client_mock.delete_subscription.mock_calls)

    def test_exists_on_non_exist_topic(self):
        """Tests topic_and_subscriptions_exist on a non-exist topic."""
        self.pub_client_mock.list_topic_subscriptions.side_effect = GaxError(
            "msg", get_rpc_error_with_status_code(StatusCode.NOT_FOUND))

        self.assertFalse(self.builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub2']))

    def test_exists_on_non_exist_sub(self):
        """Tests topic_and_subscriptions_exist on a non-exist subscription."""
        sub1_path = 'projects/myproject/subscriptions/sub1'
        sub2_path = 'projects/myproject/subscriptions/sub2'

        self.pub_client_mock.list_topic_subscriptions.return_value = [
            sub1_path,
            sub2_path,
        ]
        self.assertFalse(self.builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub3']))

    def test_exists_returns_true(self):
        """Tests topic_and_subscriptions_exist on an existing topic and
        subscriptions."""
        sub1_path = 'projects/myproject/subscriptions/sub1'
        sub2_path = 'projects/myproject/subscriptions/sub2'

        self.pub_client_mock.list_topic_subscriptions.return_value = [
            sub1_path,
            sub2_path,
        ]
        self.assertTrue(self.builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub2']))


if __name__ == '__main__':
    unittest.main()
