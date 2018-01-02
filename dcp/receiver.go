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

/*
Package dcp contains all the objects definition and the logic necessary for the
data control plane (dcp). DCP is responsible for managing the whole lifecyle of
transfers and so managing the transfer jobs and the tasks associated with them,
and provide a monitoring capabilities for the transfers.
*/
// TODO(b/63026027): Design a proper way of logging. Currently, everything is
// printed to stdout.
package dcp

import (
	"context"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
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
	Sub     gcloud.PSSubscription
	Store   Store
	Handler MessageHandler

	batcher       taskUpdatesBatcher
	jobSpecsCache struct {
		sync.RWMutex
		c *lru.Cache
	}
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

func (r *MessageReceiver) ReceiveMessages(ctx context.Context) {
	// Currently, there is a batcher for each message receiver type (list, copy).
	// Maybe we can consider only one batcher for all the receiver types.
	r.batcher.initializeAndStart(r.Store)
	r.jobSpecsCache.Lock()
	r.jobSpecsCache.c = lru.New(jobSpecsCacheMaxSize)
	r.jobSpecsCache.Unlock()

	err := r.Sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// TODO(b/63058868): Failed to handle a PubSub message will be keep
		// redelivered by the PubSub for significant amount of time (1 week).
		// Non-retriable errors should mark the task failed and ack the message.
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
	})

	if ctx.Err() != nil {
		glog.Warningf(
			"Error receiving messages for subscription %v, with context error: %v.",
			r.Sub, ctx.Err())
	}

	// The Pub/Sub client libraries already retries on retriable errors. Panic
	// here on non-retriable errors.
	if err != nil {
		// TODO(b/69918304): We should not panic in the managed service. Instead,
		// we should have an alerting technique for non-retriable errors.
		glog.Fatalf("Error receiving messages for subscription %v, with error: %v.",
			r.Sub, err)
	}
}
