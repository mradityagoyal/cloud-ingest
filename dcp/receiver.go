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

// TODO(b/63026027): Design a proper way of logging. Currently, everything is
// printed to stdout.
package dcp

import (
	"context"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
	"github.com/golang/groupcache/lru"
)

const (
	jobSpecsCacheMaxSize = 1000
)

// MessageHandler interface is used to abstract handling of various message
// types that correspond to various task types.
type MessageHandler interface {
	// HandleMessage processes a TaskCompletionMessage constructed from Pub/Sub task progress
	// message, and creates a TaskUpdate, with all expected new tasks for the next step of the
	// workflow.
	HandleMessage(jobSpec *JobSpec, taskCompletionMessage *TaskCompletionMessage) (*TaskUpdate, error)
}

// MessageReceiver receives outstanding messages from a PubSub subscription. For
// each message, it executes the corresponding handler to determine next workflow steps, and then
// passes the result to the batcher to perform transactional updates and ack the message.
// TODO(b/63014764): Add unit tests for MessageReceiver.
type MessageReceiver struct {
	PubSubClientFunc func(ctx context.Context, projectID string) (gcloud.PS, error)
	Store            Store
	Handler          MessageHandler
	Ticker           helpers.Ticker

	batcher       taskUpdatesBatcher
	jobSpecsCache struct {
		sync.RWMutex
		c *lru.Cache
	}
	failedSubs       chan string
	projectClientMap map[string]gcloud.PS
}

type SubscriptionMapGetter func() (map[string]string, error)
type PubSubClientGetter func(ctx context.Context, projectID string) (gcloud.PS, error)

func NewMessageReceiver(
	pscg PubSubClientGetter, store Store, handler MessageHandler) *MessageReceiver {
	r := MessageReceiver{
		PubSubClientFunc: pscg,
		Store:            store,
		Handler:          handler,
	}
	r.initializeBatcher()
	r.failedSubs = make(chan (string))
	r.projectClientMap = make(map[string]gcloud.PS)
	return &r
}

func (r *MessageReceiver) getJobSpec(jobConfigRRStruct JobConfigRRStruct) (*JobSpec, error) {
	// Try to find the job from the cache.
	r.jobSpecsCache.RLock()
	jobSpec, ok := r.jobSpecsCache.c.Get(jobConfigRRStruct)
	r.jobSpecsCache.RUnlock()
	if ok {
		return jobSpec.(*JobSpec), nil
	}
	// TODO(b/69675852): Multiple threads will get the reader lock, all of them
	// will try to get the job spec from the store and update it in the cache.
	// This is unlikely because the list task will be probably the first one to
	// cache the job spec, and all the other tasks are dependent on it.

	// Get the job spec from the store and add it to the cache.
	glog.Infof("Did not find Job Spec for (%s) in the cache, querying the store",
		jobConfigRRStruct)
	storeJobSpec, err := r.Store.GetJobSpec(jobConfigRRStruct)
	if err != nil {
		return nil, err
	}
	r.jobSpecsCache.Lock()
	r.jobSpecsCache.c.Add(jobConfigRRStruct, storeJobSpec)
	r.jobSpecsCache.Unlock()
	return storeJobSpec, nil
}

func (r *MessageReceiver) SingleSubReceiveMessages(ctx context.Context, sub gcloud.PSSubscription) {
	r.receiveMessages(ctx, sub, "", false)
}

func (r *MessageReceiver) RoundRobinReceiveMessages(
	ctx context.Context, subMapGetter SubscriptionMapGetter) {
	// TODO (b/71647771): PubSub Go client currently doesn't support an easy way to create a
	// subscription outside of the Client struct's project.  When
	// https://github.com/GoogleCloudPlatform/google-cloud-go/issues/849 is fixed,
	// drop the one-client-per-subscription.
	select {
	case failedProjectID := <-r.failedSubs:
		// The subscription for this project failed; recreate it and retry.
		r.projectClientMap[failedProjectID] = nil
	default:
		m, err := subMapGetter()
		if err != nil {
			// TODO (b/72335955): Instead of dying here and letting the DCP retry when it
			// is restarted, continue monitoring the subscriptions that we have and
			// add monitoring so that we can alert that we're unable to retrieve from
			// the store.
			glog.Fatalf("Retrieving ProjectID:Subscription failed, error: %v", err)
		}

		for projectID, subscriptionID := range m {
			if r.projectClientMap[projectID] != nil {
				// There is already a goroutine receiving messages for this project's subscription.
				continue
			}
			pubSubClient, err := r.PubSubClientFunc(ctx, projectID)
			if err != nil {
				glog.Warningf(
					"Could not create PubSub client for project %s, error: %v.", projectID, err)
				continue
			}
			r.projectClientMap[projectID] = pubSubClient
			sub := pubSubClient.Subscription(subscriptionID)
			// TODO (b/71648278): Add a leasing mechanism so that this scales beyond the number
			// of listeners that can run in a single DCP.
			go r.receiveMessages(ctx, sub, projectID, true)
		}
	}
}

func (r *MessageReceiver) initializeBatcher() {
	// Currently, there is a batcher for each message receiver type (list, copy).
	// Maybe we can consider only one batcher for all the receiver types.
	r.batcher.initializeAndStart(r.Store)
	r.jobSpecsCache.Lock()
	r.jobSpecsCache.c = lru.New(jobSpecsCacheMaxSize)
	r.jobSpecsCache.Unlock()
}

func (r *MessageReceiver) receiveMessages(
	ctx context.Context, sub gcloud.PSSubscription, projectID string, reportFailedSubs bool) {
	// TODO(b/63058868): Failed to handle a PubSub message will be keep
	// redelivered by the PubSub for significant amount of time (1 week).
	// Non-retriable errors should mark the task failed and ack the message.
	err := sub.Receive(ctx, r.subReceiveFunc)

	if ctx.Err() != nil {
		glog.Warningf(
			"Context for receiveMessages on sub %v, was cancelled with context error: %v.",
			sub, ctx.Err())
	}

	// The Pub/Sub client libraries already retries on retriable errors. Panic
	// here on non-retriable errors.
	if err != nil {
		glog.Warningf("Error receiving messages for subscription %v, with error: %v.",
			sub, err)
	}
	if reportFailedSubs {
		r.failedSubs <- projectID
	}
}

func (r *MessageReceiver) subReceiveFunc(ctx context.Context, msg *pubsub.Message) {
	glog.Infof("Handling a message: %s.", string(msg.Data))
	taskCompletionMessage, err := TaskCompletionMessageFromJson(msg.Data)
	if err != nil {
		glog.Errorf("Error handling the message: %s with error: %v.",
			string(msg.Data), err)
		return
	}
	taskRRStruct, err := TaskRRStructFromTaskRRName(taskCompletionMessage.TaskRRName)
	if err != nil {
		glog.Errorf("Error getting JobConfigID from TaskIDStr %s: %v",
			taskCompletionMessage.TaskRRName, err)
		return
	}
	jobSpec, err := r.getJobSpec(taskRRStruct.JobConfigRRStruct)
	if err != nil {
		glog.Errorf("Error in getting JobSpec of JobConfig: %v, with error: %v.",
			taskRRStruct.JobConfigRRStruct, err)
		return
	}

	taskUpdate, err := r.Handler.HandleMessage(jobSpec, taskCompletionMessage)
	if err != nil {
		glog.Errorf(
			"Error handling the message: %s, with job spec: %v, and taskCompletionMessage: %v: %v",
			string(msg.Data), jobSpec, taskCompletionMessage, err)
		return
	}

	r.batcher.addTaskUpdate(taskUpdate, msg)
}
