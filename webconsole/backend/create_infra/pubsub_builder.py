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

# pylint: disable=relative-import
import google.auth as googleauth

from google.cloud import pubsub
from google.gax import GaxError
from google.gax import config
from grpc import StatusCode

from proto.tasks_pb2 import ResourceStatus

class PubSubBuilder(object):
    """Manipulates PubSub topics/subscriptions."""

    def __init__(self, credentials=None, project_id=None):
        self.project_id = project_id
        if not self.project_id:
            _, self.project_id = googleauth.default()

        self.pub_client = pubsub.PublisherClient(credentials=credentials)
        self.sub_client = pubsub.SubscriberClient(credentials=credentials)

    def create_topic(self, topic_name):
        """Creates topic_name topic."""
        full_topic_name = self.pub_client.topic_path(
            self.project_id, topic_name)
        self.pub_client.create_topic(full_topic_name)

    def create_subscription(self, topic_name, sub_name, ack_deadline=30):
        """Creates sub_name subscription in topic with deadline ack_deadline."""
        full_topic_name = self.sub_client.topic_path(
            self.project_id, topic_name)
        full_sub_name = self.sub_client.subscription_path(
            self.project_id, sub_name)
        self.sub_client.create_subscription(full_sub_name, full_topic_name,
                                            ack_deadline_seconds=ack_deadline)

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
        full_topic_name = self.pub_client.topic_path(
            self.project_id, topic_name)

        topic_subs_set = set()

        try:
            for sub in (self.pub_client.
                        list_topic_subscriptions(full_topic_name)):
                topic_subs_set.add(sub)
        except GaxError as err:
            if config.exc_to_code(err.cause) != StatusCode.NOT_FOUND:
                raise
            return False
        for sub in sub_names:
            full_sub_name = self.sub_client.subscription_path(
                self.project_id, sub)
            if full_sub_name not in topic_subs_set:
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

    def create_topic_and_subscriptions(self, topic_name, sub_names,
                                       ack_deadline=30):
        """Creates topic_name topics and associate sub_names subscriptions."""
        self.create_topic(topic_name)
        for sub_name in sub_names:
            self.create_subscription(topic_name, sub_name,
                                     ack_deadline=ack_deadline)

    def delete_topic_and_subscriptions(self, topic_name):
        """Deletes topic_name topic and its subscriptions."""
        full_topic_name = self.pub_client.topic_path(
            self.project_id, topic_name)
        try:
            for sub in (self.pub_client.
                        list_topic_subscriptions(full_topic_name)):
                self.sub_client.delete_subscription(sub)

        except GaxError as err:
            if config.exc_to_code(err.cause) != StatusCode.NOT_FOUND:
                raise
            print 'Topic {} does not exist, skipping delete.'.format(topic_name)
            return

        self.pub_client.delete_topic(full_topic_name)
