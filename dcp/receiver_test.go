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

package dcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/groupcache/lru"
	"github.com/golang/mock/gomock"
)

func TestGetJobSpecFromCache(t *testing.T) {
	initialJobSpec := &JobSpec{
		OnpremSrcDirectory: "dummy-src",
		GCSBucket:          "dummy-bucket",
	}
	store := &FakeStore{
		jobSpec: initialJobSpec,
	}
	r := MessageReceiver{
		Store: store,
	}
	r.jobSpecsCache.c = lru.New(1)

	// The first call to getJobSepc puts the job spec in the cache.
	jobConfigRRStruct := JobConfigRRStruct{"dummy-project", "dummy-config"}
	jobSpec, _ := r.getJobSpec(jobConfigRRStruct)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}

	// Change the stored job from the cached spec and make sure the
	// MessageReceiver gets it from cache.
	store.jobSpec = &JobSpec{
		OnpremSrcDirectory: "modified-dummy-src",
		GCSBucket:          "modified-dummy-bucket",
	}

	// The second call should get the job spec from the cache.
	jobSpec, _ = r.getJobSpec(jobConfigRRStruct)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}
}

func TestGetJobSpecThatRemovedFromCache(t *testing.T) {
	initialJobSpec := &JobSpec{
		OnpremSrcDirectory: "dummy-src",
		GCSBucket:          "dummy-bucket",
	}
	store := &FakeStore{
		jobSpec: initialJobSpec,
	}
	r := MessageReceiver{
		Store: store,
	}
	r.jobSpecsCache.c = lru.New(1)

	// The first call to getJobSepc puts the job spec in the cache.
	jobConfigRRStruct1 := JobConfigRRStruct{"dummy-project", "dummy-config-1"}
	jobSpec, _ := r.getJobSpec(jobConfigRRStruct1)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}

	// Add another item in the cache so the first one got removed.
	jobConfigRRStruct2 := JobConfigRRStruct{"dummy-project", "dummy-config-2"}
	r.getJobSpec(jobConfigRRStruct2)

	// Change the stored job from the cached spec and make sure the
	// MessageReceiver gets it from cache.
	storedJobSpec := &JobSpec{
		OnpremSrcDirectory: "modified-dummy-src",
		GCSBucket:          "modified-dummy-bucket",
	}
	store.jobSpec = storedJobSpec

	// Reading the removed job spec should come from the store, not the cache.
	jobSpec, _ = r.getJobSpec(jobConfigRRStruct1)
	if !reflect.DeepEqual(jobSpec, storedJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}
}

func TestRoundRobinReceiveMessagesFallbackSub(t *testing.T) {
	// Tests that the round-robin receiver uses defaults if the store has no subscription IDs.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListSub := gcloud.NewMockPSSubscription(mockCtrl)
	store := &FakeStore{listProgSubMap: map[string]string{}}
	ticker := helpers.NewMockTicker()
	r := MessageReceiver{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			if projectID != "fallbackProjectID" {
				t.Errorf("Expected pubsub client for 'fallbackProjectID', got '%s'", projectID)
			}
			return mockPubSub, nil
		},
		Store:  store,
		Ticker: ticker,
	}
	mockPubSub.EXPECT().Subscription("fallbackSubID").MaxTimes(1).Return(mockListSub)
	mockListSub.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	ctx, stopRunner := context.WithCancel(context.Background())
	go r.RoundRobinReceiveMessages(
		ctx,
		store.GetListProgressSubscriptionsMap,
		"fallbackProjectID",
		"fallbackSubID")
	ticker.Tick()
	stopRunner()
}

func TestRoundRobinReceiveMessagesMultipleSubs(t *testing.T) {
	// Tests that the round-robin receiver accepts multiple projects and subscriptions.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListSub1 := gcloud.NewMockPSSubscription(mockCtrl)
	mockListSub2 := gcloud.NewMockPSSubscription(mockCtrl)
	store := &FakeStore{}
	store.AddListSubscription("fakeProjectID1", "fakeListSubID1")
	store.AddListSubscription("fakeProjectID2", "fakeListSubID2")
	ticker := helpers.NewMockTicker()
	r := MessageReceiver{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store:  store,
		Ticker: ticker,
	}
	mockPubSub.EXPECT().Subscription("fakeListSubID1").MaxTimes(1).Return(mockListSub1)
	mockPubSub.EXPECT().Subscription("fakeListSubID2").MaxTimes(1).Return(mockListSub2)
	mockListSub1.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	mockListSub2.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	ctx, stopRunner := context.WithCancel(context.Background())
	go r.RoundRobinReceiveMessages(
		ctx,
		store.GetListProgressSubscriptionsMap,
		"fallbackProjectID",
		"fallbackSubID")
	ticker.Tick()
	stopRunner()
}

func TestRoundRobinReceiveMessagesSubDies(t *testing.T) {
	// Tests that subscriptions are recreated in the round-robin receiver when they die.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListSub := gcloud.NewMockPSSubscription(mockCtrl)
	mockListSubRecreated := gcloud.NewMockPSSubscription(mockCtrl)
	store := &FakeStore{}
	store.AddListSubscription("fakeProjectID1", "fakeListSubID1")
	ticker := helpers.NewMockTicker()
	r := MessageReceiver{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store:  store,
		Ticker: ticker,
	}
	callNumber := 0
	mockPubSub.EXPECT().Subscription("fakeListSubID1").DoAndReturn(
		func(projectID string) *gcloud.MockPSSubscription {
			if callNumber == 0 {
				callNumber++
				return mockListSub
			} else {
				return mockListSubRecreated
			}
		}).MaxTimes(2)
	mockListSub.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1).Return(
		errors.New("sub died"))
	mockListSubRecreated.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	ctx, stopRunner := context.WithCancel(context.Background())
	go r.RoundRobinReceiveMessages(
		ctx,
		store.GetListProgressSubscriptionsMap,
		"fallbackProjectID",
		"fallbackSubID")
	ticker.Tick() // First iteration: Subscription.Receive dies
	ticker.Tick() // Second iteration: Subscription.Receive succeeds
	stopRunner()
}

func TestRoundRobinReceiveMessagesNewSub(t *testing.T) {
	// Tests that new subscriptions are picked up by the round-robin runner.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListSub1 := gcloud.NewMockPSSubscription(mockCtrl)
	mockListSub2 := gcloud.NewMockPSSubscription(mockCtrl)
	store := &FakeStore{}
	store.AddListSubscription("fakeProjectID1", "fakeListSubID1")
	ticker := helpers.NewMockTicker()
	r := MessageReceiver{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store:  store,
		Ticker: ticker,
	}
	mockPubSub.EXPECT().Subscription("fakeListSubID1").MaxTimes(1).Return(mockListSub1)
	mockPubSub.EXPECT().Subscription("fakeListSubID2").MaxTimes(1).Return(mockListSub2)
	mockListSub1.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	mockListSub2.EXPECT().Receive(gomock.Any(), gomock.Any()).MaxTimes(1)
	ctx, stopRunner := context.WithCancel(context.Background())
	go r.RoundRobinReceiveMessages(
		ctx,
		store.GetListProgressSubscriptionsMap,
		"fallbackProjectID",
		"fallbackSubID")
	ticker.Tick() // First iteration: First subscription is live
	store.AddListSubscription("fakeProjectID2", "fakeListSubID2")
	// Add second subscription. Depending on timing, this could get picked up during the second
	// or third iterations.
	ticker.Tick()
	ticker.Tick()
	stopRunner()
}
