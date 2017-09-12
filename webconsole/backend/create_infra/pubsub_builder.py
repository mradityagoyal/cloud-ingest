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
# limitations under the License.

"""Google Cloud PubSub admin utilities."""

from google.cloud import exceptions
from google.cloud import pubsub

from resource_status import ResourceStatus

class PubSubBuilder(object):
    """Manipulates PubSub topics/subscriptions."""

    def __init__(self, client=None):
        self.client = client
        if not self.client:
            # Use the default client if not specified.
            self.client = pubsub.Client()

    def create_topic(self, topic_name):
        """Creates topic_name topic."""
        topic = self.client.topic(topic_name)
        topic.create()
        return topic

    @classmethod
    def create_subscription(cls, topic, sub_name, ack_deadline=15):
        """Creates sub_name subscription in topic with deadline ack_deadline."""
        sub = topic.subscription(sub_name, ack_deadline=ack_deadline)
        sub.create()

    def topic_and_subscriptions_exist(self, topic_name, sub_names):
        """Checks the existence of a topic and associated subscriptions.

        Checks whether a topic_name and the associated sub_names subscriptions
        exist.

        Args:
            topic_name: Name of the topic.
            sub_names: Array of the subscriptions names.

        Returns:
            True if the topic and all the subscriptions exist.
        """
        topic = self.client.topic(topic_name)
        if not topic.exists():
            return False
        for sub_name in sub_names:
            if not topic.subscription(sub_name).exists():
                return False
        return True

    def topic_and_subscriptions_status(self, topic_name, sub_name):
        """Gets status of of a topic and associated subscriptions.

        Args:
            topic_name: Name of the topic.
            sub_names: Array of the subscriptions names.

        Returns:
            ResourceStatus enum of the status of the topic and subscriptions.
        """
        if self.topic_and_subscriptions_exist(topic_name, sub_name):
            return ResourceStatus.RUNNING
        return ResourceStatus.NOT_FOUND

    def create_topic_and_subscriptions(self, topic_name, sub_names):
        """Creates topic_name topics and associate sub_names subscriptions."""
        topic = self.create_topic(topic_name)
        for sub_name in sub_names:
            self.create_subscription(topic, sub_name)

    def delete_topic_and_subscriptions(self, topic_name):
        """Deletes topic_name topic and its subscriptions."""
        topic = self.client.topic(topic_name)
        if not topic.exists():
            print 'Topic {} does not exist, skipping delete.'.format(topic.name)
            return

        # Deleting subscriptions associated with the topic
        subs = topic.list_subscriptions()
        for sub in subs:
            try:
                sub.delete()
            except exceptions.NotFound:
                print 'Subscription {} does not exist, skipping delete.'.format(
                    sub.name)

        # Delete the topic
        topic.delete()
