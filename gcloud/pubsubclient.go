/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gcloud

import (
	"cloud.google.com/go/pubsub"
	"context"
)

// Minimal wrapper around PubSub client.

type PS interface {
	Topic(id string) PSTopic
	TopicInProject(id, projectID string) PSTopic
}

type PSTopic interface {
	Publish(ctx context.Context, msg *pubsub.Message) *pubsub.PublishResult
	Stop()
}

type PSSubscription interface {
	Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error
}

type PubSubClient struct {
	client *pubsub.Client
}

type PubSubTopic struct {
	topic *pubsub.Topic
}

type PubSubSubscription struct {
	topic *pubsub.Topic
}

func NewPubSubClient(client *pubsub.Client) *PubSubClient {
	return &PubSubClient{client}
}

func (c *PubSubClient) NewPubSubClient(ctx context.Context, database string) (*pubsub.Client, error) {
	return pubsub.NewClient(ctx, database)
}

func (c *PubSubClient) TopicInProject(id, projectID string) PSTopic { // *pubsub.Topic {
	return c.client.TopicInProject(id, projectID)
}

func (c *PubSubClient) Topic(id string) PSTopic { // pubsub.Topic {
	return c.client.Topic(id)
}

func (t *PubSubTopic) Publish(ctx context.Context, msg *pubsub.Message) *pubsub.PublishResult {
	return t.Publish(ctx, msg)
}

func (t *PubSubTopic) Stop() {
	t.Stop()
}

func (c *PubSubClient) Subscription(id string) PSSubscription {
	return c.client.Subscription(id)
}

func (s *PubSubSubscription) Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error {
	return s.Receive(ctx, f)
}
