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

# pylint: disable=import-error,no-name-in-module
from google.cloud import exceptions
from google.cloud import pubsub
from mock import MagicMock
from mock import call
from mock import patch

from pubsub_builder import PubSubBuilder


# pylint: disable=too-many-public-methods
class PubsubBuilderTest(unittest.TestCase):
    """Unit tests for pubsub_builder.py"""

    def setUp(self):
        self.default_client = MagicMock()
        self.patcher = patch.object(pubsub, 'Client',
                                    return_value=self.default_client)
        self.patcher.start()

    def tearDown(self):
        self.patcher.stop()

    def test_init_with_default_client(self):
        """Tests create PubSubBuilder with default client."""
        builder = PubSubBuilder()
        self.assertEqual(builder.client, self.default_client)

    def test_init_with_client(self):
        """Tests create PubSubBuilder with passed client."""
        client_mock = MagicMock()
        builder = PubSubBuilder(client_mock)
        self.assertEqual(builder.client, client_mock)

    def test_create_topic(self):
        """Tests creating a pubsub topic."""
        builder = PubSubBuilder()
        topic = builder.create_topic('mytopic')
        self.default_client.topic.assert_called_once_with('mytopic')
        topic.create.assert_called_once_with()

    @classmethod
    def test_create_subscription(cls):
        """Tests creating a topic subscription."""
        topic_mock = MagicMock()
        sub_mock = MagicMock()
        topic_mock.subscription.return_value = sub_mock

        PubSubBuilder.create_subscription(topic_mock, 'mysub', ack_deadline=30)
        topic_mock.subscription.assert_called_once_with('mysub',
                                                        ack_deadline=30)
        sub_mock.create.assert_called_once_with()

    def test_create_topic_subs(self):
        """Tests creating a pubsub topic and associated subscriptions."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()

        builder.create_topic = MagicMock()
        builder.create_topic.return_value = topic_mock
        builder.create_subscription = MagicMock()

        builder.create_topic_and_subscriptions('mytopic', ['sub1', 'sub2'])

        builder.create_topic.assert_called_once_with('mytopic')
        expected_calls = [
            call(topic_mock, 'sub1'),
            call(topic_mock, 'sub2')
        ]
        self.assertItemsEqual(expected_calls,
                              builder.create_subscription.mock_calls)

    def test_delete_non_exists_topic(self):
        """Tests deleting a non-exists pubsub topic."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()
        self.default_client.topic.return_value = topic_mock
        topic_mock.exists.return_value = False

        builder.delete_topic_and_subscriptions('mytopic')
        self.default_client.topic.assert_called_once_with('mytopic')
        topic_mock.list_subscriptions.assert_not_called()

    def test_delete_topic_subs(self):
        """Tests deleting a pubsub topic with all of its subscriptions."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()
        self.default_client.topic.return_value = topic_mock
        topic_mock.exists.return_value = True

        sub_mock_1 = MagicMock()
        sub_mock_2 = MagicMock()
        sub_mock_1.delete.side_effect = exceptions.NotFound(
            'Not found subscription.')
        topic_mock.list_subscriptions.return_value = (sub_mock_1, sub_mock_2)

        builder.delete_topic_and_subscriptions('mytopic')
        self.default_client.topic.assert_called_once_with('mytopic')

        topic_mock.list_subscriptions.assert_called_once_with()
        sub_mock_1.delete.assert_called_once_with()
        sub_mock_2.delete.assert_called_once_with()
        topic_mock.delete.assert_called_once_with()

    def test_exists_on_non_exist_topic(self):
        """Tests topic_and_subscriptions_exist on a non-exist topic."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()
        self.default_client.topic.return_value = topic_mock
        topic_mock.exists.return_value = False

        self.assertFalse(builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub2']))

    def test_exists_on_non_exist_sub(self):
        """Tests topic_and_subscriptions_exist on a non-exist subscription."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()
        self.default_client.topic.return_value = topic_mock
        topic_mock.exists.return_value = True

        sub_mock_1 = MagicMock()
        sub_mock_1.exists.return_value = True
        sub_mock_2 = MagicMock()
        sub_mock_2.exists.return_value = False

        topic_mock.subscriptions.side_effect = [sub_mock_1, sub_mock_2]
        self.assertFalse(builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub2']))

    def test_exists_returns_true(self):
        """Tests topic_and_subscriptions_exist on an existing topic and
        subscriptions."""
        builder = PubSubBuilder()
        topic_mock = MagicMock()
        self.default_client.topic.return_value = topic_mock
        topic_mock.exists.return_value = True

        subscriptions_mock = MagicMock()
        topic_mock.subscriptions = subscriptions_mock

        sub_mock_1 = MagicMock()
        sub_mock_1.exists.return_value = True
        sub_mock_2 = MagicMock()
        sub_mock_2.exists.return_value = True

        subscriptions_mock.side_effect = [sub_mock_1, sub_mock_2]
        self.assertTrue(builder.topic_and_subscriptions_exist(
            'mytopic', ['sub1', 'sub2']))
        self.assertItemsEqual(subscriptions_mock.mock_calls,
                              [call('sub1'), call('sub2')])


if __name__ == '__main__':
    unittest.main()
