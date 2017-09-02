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

# pylint: disable=import-error,no-name-in-module
from google.cloud import exceptions
from google.cloud import pubsub


class PubSubBuilder(object):
    """Manipulates PubSub topics/subscriptions."""

    def __init__(self):
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
