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
	Subscription(id string) PSSubscription
}

type PSTopic interface {
	Publish(ctx context.Context, msg *pubsub.Message) PSPublishResult
	Stop()
	ID() string
	Delete(ctx context.Context) error
	Exists(ctx context.Context) (bool, error)
}

type PSSubscription interface {
	Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error
	ID() string
	Delete(ctx context.Context) error
	Config(ctx context.Context) (PSSubscriptionConfig, error)
	Exists(ctx context.Context) (bool, error)
}

type PSPublishResult interface {
	Get(ctx context.Context) (serverID string, err error)
}

type PSSubscriptionConfig interface {
	Topic() PSTopic
}

type PubSubClient struct {
	client *pubsub.Client
}

type PubSubTopic struct {
	topic *pubsub.Topic
}

type PubSubSubscription struct {
	sub *pubsub.Subscription
}

type PubSubPublishResult struct {
	result *pubsub.PublishResult
}

type PubSubTopicWrapper struct {
	topic *pubsub.Topic
}

type PubSubSubscriptionWrapper struct {
	sub *pubsub.Subscription
}

type PubSubSubscriptionConfig struct {
	topic PSTopic
}

func NewPubSubClient(client *pubsub.Client) *PubSubClient {
	return &PubSubClient{client}
}

func (c *PubSubClient) NewPubSubClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	return pubsub.NewClient(ctx, projectID)
}

// NewPubSubTopicWrapper wraps a pubsub.Topic to ensure its Publish function can return mock results
func NewPubSubTopicWrapper(t *pubsub.Topic) *PubSubTopicWrapper {
	return &PubSubTopicWrapper{t}
}

func (w *PubSubTopicWrapper) Publish(ctx context.Context, msg *pubsub.Message) PSPublishResult {
	return w.topic.Publish(ctx, msg)
}

func (w *PubSubTopicWrapper) Stop() {
	w.topic.Stop()
}

func (w *PubSubTopicWrapper) ID() string {
	return w.topic.ID()
}

func (w *PubSubTopicWrapper) Delete(ctx context.Context) error {
	return w.topic.Delete(ctx)
}

func (w *PubSubTopicWrapper) Exists(ctx context.Context) (bool, error) {
	return w.topic.Exists(ctx)
}

// NewPubSubSubscriptionWrapper wraps a pubsub.Subscription to ensure its Config function
// can return mock results
func NewPubSubSubscriptionWrapper(s *pubsub.Subscription) *PubSubSubscriptionWrapper {
	return &PubSubSubscriptionWrapper{s}
}

func (w *PubSubSubscriptionWrapper) Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error {
	return w.sub.Receive(ctx, f)
}

func (w *PubSubSubscriptionWrapper) ID() string {
	return w.sub.ID()
}

func (w *PubSubSubscriptionWrapper) Delete(ctx context.Context) error {
	return w.sub.Delete(ctx)
}

func (w *PubSubSubscriptionWrapper) Config(ctx context.Context) (PSSubscriptionConfig, error) {
	config, err := w.sub.Config(ctx)
	if err != nil {
		return nil, err
	}
	return &PubSubSubscriptionConfig{topic: NewPubSubTopicWrapper(config.Topic)}, nil
}

func (w *PubSubSubscriptionWrapper) Exists(ctx context.Context) (bool, error) {
	return w.sub.Exists(ctx)
}

func (c *PubSubClient) TopicInProject(id, projectID string) PSTopic {
	return NewPubSubTopicWrapper(c.client.TopicInProject(id, projectID))
}

func (c *PubSubClient) Topic(id string) PSTopic {
	return NewPubSubTopicWrapper(c.client.Topic(id))
}

func (t *PubSubTopic) Publish(ctx context.Context, msg *pubsub.Message) PSPublishResult {
	return t.topic.Publish(ctx, msg)
}

func (t *PubSubTopic) Stop() {
	t.topic.Stop()
}

func (t *PubSubTopic) ID() string {
	return t.topic.ID()
}

func (t *PubSubTopic) Delete(ctx context.Context) error {
	return t.topic.Delete(ctx)
}

func (t *PubSubTopic) Exists(ctx context.Context) (bool, error) {
	return t.topic.Exists(ctx)
}

func (c *PubSubClient) Subscription(id string) PSSubscription {
	return NewPubSubSubscriptionWrapper(c.client.Subscription(id))
}

func (s *PubSubSubscription) Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error {
	return s.sub.Receive(ctx, f)
}

func (s *PubSubSubscription) ID() string {
	return s.sub.ID()
}

func (s *PubSubSubscription) Delete(ctx context.Context) error {
	return s.sub.Delete(ctx)
}

func (s *PubSubSubscription) Config(ctx context.Context) (PSSubscriptionConfig, error) {
	config, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	return &PubSubSubscriptionConfig{topic: config.Topic()}, nil
}

func (s *PubSubSubscription) Exists(ctx context.Context) (bool, error) {
	return s.sub.Exists(ctx)
}

func (p *PubSubPublishResult) Get(ctx context.Context) (serverID string, err error) {
	return p.result.Get(ctx)
}

func NewPubSubSubscriptionConfig(topic PSTopic) PSSubscriptionConfig {
	return &PubSubSubscriptionConfig{topic: topic}
}

func (c *PubSubSubscriptionConfig) Topic() PSTopic {
	return c.topic
}
