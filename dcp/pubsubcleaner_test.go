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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/mock/gomock"
)

func TestCleanPubSubNoProjects(t *testing.T) {
	// Test that the cleaner does not call PubSub when there are no unused projects.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{unusedProjects: make([]*ProjectInfo, 0)}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	p.CleanPubSub()
}

func TestCleanPubSubWithProject(t *testing.T) {
	// Test that the cleaner deletes an unused project.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{
		unusedProjects: []*ProjectInfo{
			&ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
		},
	}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)
	mockCopyProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)

	mockPubSub.EXPECT().Topic("lt1").Return(mockListTopic)
	mockPubSub.EXPECT().Topic("ct1").Return(mockCopyTopic)
	mockPubSub.EXPECT().Topic("lpt1").Return(mockListProgressTopic)
	mockPubSub.EXPECT().Topic("cpt1").Return(mockCopyProgressTopic)
	mockPubSub.EXPECT().Subscription("ls1").Return(mockListProgressSubscription)
	mockPubSub.EXPECT().Subscription("cs1").Return(mockCopyProgressSubscription)

	mockListProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockListProgressTopic), nil)
	mockListProgressTopic.EXPECT().ID().Return("lpt1")
	mockCopyProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockCopyProgressTopic), nil)
	mockCopyProgressTopic.EXPECT().ID().Return("cpt1")

	mockListTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)

	mockListTopic.EXPECT().Delete(gomock.Any())
	mockCopyTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressTopic.EXPECT().Delete(gomock.Any())
	mockListProgressTopic.EXPECT().Delete(gomock.Any())
	mockListProgressSubscription.EXPECT().Delete(gomock.Any())
	mockCopyProgressSubscription.EXPECT().Delete(gomock.Any())

	p.CleanPubSub()
	remainingProjects, err := store.GetUnusedProjects(1)
	if err != nil {
		t.Errorf("could not retreive unused projects, error: %v", err)
	}
	if len(remainingProjects) != 0 {
		t.Errorf("expected 0 remaining projects, but got %d", len(remainingProjects))
	}
}

func TestCleanPubSubSomeNotExist(t *testing.T) {
	// Tests the pubsub cleaner when some topics and subscriptions don't exist.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{
		unusedProjects: []*ProjectInfo{
			&ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
		},
	}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	// Copy topic doesn't exist for this test.
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	// List progress subscription doesn't exist, so it is assumed that the topic is also gone.
	mockListProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)
	mockCopyProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)

	mockPubSub.EXPECT().Topic("lt1").Return(mockListTopic)
	mockPubSub.EXPECT().Topic("ct1").Return(mockCopyTopic)
	mockPubSub.EXPECT().Topic("cpt1").Return(mockCopyProgressTopic)
	mockPubSub.EXPECT().Subscription("ls1").Return(mockListProgressSubscription)
	mockPubSub.EXPECT().Subscription("cs1").Return(mockCopyProgressSubscription)

	mockCopyProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockCopyProgressTopic), nil)
	mockCopyProgressTopic.EXPECT().ID().Return("cpt1")

	mockListTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyTopic.EXPECT().Exists(gomock.Any()).Return(false, nil)
	mockListProgressSubscription.EXPECT().Exists(gomock.Any()).Return(false, nil)
	mockCopyProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)

	mockListTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressSubscription.EXPECT().Delete(gomock.Any())

	p.CleanPubSub()
	remainingProjects, err := store.GetUnusedProjects(1)
	if err != nil {
		t.Errorf("could not retreive unused projects, error: %v", err)
	}
	if len(remainingProjects) != 0 {
		t.Errorf("expected 0 remaining projects, but got %d", len(remainingProjects))
	}
}

func TestCleanPubSubAllNotExist(t *testing.T) {
	// Test that the cleaner deletes an unused project when all of the topics/subs don't exist.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{
		unusedProjects: []*ProjectInfo{
			&ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
		},
	}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	// Subscriptions don't exist, so the topics will also not exist.
	mockListProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)
	mockCopyProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)

	mockPubSub.EXPECT().Topic("lt1").Return(mockListTopic)
	mockPubSub.EXPECT().Topic("ct1").Return(mockCopyTopic)
	mockPubSub.EXPECT().Subscription("ls1").Return(mockListProgressSubscription)
	mockPubSub.EXPECT().Subscription("cs1").Return(mockCopyProgressSubscription)

	mockListTopic.EXPECT().Exists(gomock.Any()).Return(false, nil)
	mockCopyTopic.EXPECT().Exists(gomock.Any()).Return(false, nil)
	mockListProgressSubscription.EXPECT().Exists(gomock.Any()).Return(false, nil)
	mockCopyProgressSubscription.EXPECT().Exists(gomock.Any()).Return(false, nil)

	p.CleanPubSub()
	remainingProjects, err := store.GetUnusedProjects(1)
	if err != nil {
		t.Errorf("could not retreive unused projects, error: %v", err)
	}
	if len(remainingProjects) != 0 {
		t.Errorf("expected 0 remaining projects, but got %d", len(remainingProjects))
	}
}

func TestCleanPubSubErrorDeleting(t *testing.T) {
	// Tests that the cleaner does not delete an unused project if there was an
	// error deleting a topic/sub.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{
		unusedProjects: []*ProjectInfo{
			&ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
		},
	}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)
	mockCopyProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)

	mockPubSub.EXPECT().Topic("lt1").Return(mockListTopic)
	mockPubSub.EXPECT().Topic("ct1").Return(mockCopyTopic)
	mockPubSub.EXPECT().Topic("lpt1").Return(mockListProgressTopic)
	mockPubSub.EXPECT().Topic("cpt1").Return(mockCopyProgressTopic)
	mockPubSub.EXPECT().Subscription("ls1").Return(mockListProgressSubscription)
	mockPubSub.EXPECT().Subscription("cs1").Return(mockCopyProgressSubscription)

	mockListProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockListProgressTopic), nil)
	mockListProgressTopic.EXPECT().ID().Return("lpt1")
	mockCopyProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockCopyProgressTopic), nil)
	mockCopyProgressTopic.EXPECT().ID().Return("cpt1")

	mockListTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)

	mockListTopic.EXPECT().Delete(gomock.Any()).Return(errors.New("some error"))
	mockCopyTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressTopic.EXPECT().Delete(gomock.Any())
	mockListProgressTopic.EXPECT().Delete(gomock.Any())
	mockListProgressSubscription.EXPECT().Delete(gomock.Any())
	mockCopyProgressSubscription.EXPECT().Delete(gomock.Any())

	p.CleanPubSub()
	remainingProjects, err := store.GetUnusedProjects(1)
	if err != nil {
		t.Errorf("could not retreive unused projects, error: %v", err)
	}
	if len(remainingProjects) != 1 {
		t.Errorf("expected 1 remaining projects, but got %d", len(remainingProjects))
	}
}

func TestCleanPubSubErrorRetrieving(t *testing.T) {
	// Tests that the cleaner does not delete an unused project if there was an
	// error retrieving a topic/sub.
	// Also tests that if a topic cannot be retrieved from a subscription that
	// the subscription is not deleted, so that the topic can be retrieved in
	// the future.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	store := &FakeStore{
		unusedProjects: []*ProjectInfo{
			&ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
		},
	}
	p := PubSubCleaner{
		PubSubClientFunc: func(ctx context.Context, projectID string) (gcloud.PS, error) {
			return mockPubSub, nil
		},
		Store: store,
	}

	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyProgressTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockListProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)
	mockCopyProgressSubscription := gcloud.NewMockPSSubscription(mockCtrl)

	mockPubSub.EXPECT().Topic("lt1").Return(mockListTopic)
	mockPubSub.EXPECT().Topic("ct1").Return(mockCopyTopic)
	mockPubSub.EXPECT().Topic("lpt1").Return(mockListProgressTopic)
	mockPubSub.EXPECT().Topic("cpt1").Return(mockCopyProgressTopic)
	mockPubSub.EXPECT().Subscription("ls1").Return(mockListProgressSubscription)
	mockPubSub.EXPECT().Subscription("cs1").Return(mockCopyProgressSubscription)

	mockListProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockListProgressTopic), nil)
	mockListProgressTopic.EXPECT().ID().Return("lpt1")
	mockCopyProgressSubscription.EXPECT().Config(gomock.Any()).Return(
		gcloud.NewPubSubSubscriptionConfig(mockCopyProgressTopic), nil)
	mockCopyProgressTopic.EXPECT().ID().Return("cpt1")

	mockListTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressTopic.EXPECT().Exists(gomock.Any()).Return(false, errors.New("some error"))
	mockCopyProgressTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyTopic.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockListProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)
	mockCopyProgressSubscription.EXPECT().Exists(gomock.Any()).Return(true, nil)

	mockListTopic.EXPECT().Delete(gomock.Any())
	mockCopyTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressTopic.EXPECT().Delete(gomock.Any())
	mockCopyProgressSubscription.EXPECT().Delete(gomock.Any())

	p.CleanPubSub()
	remainingProjects, err := store.GetUnusedProjects(1)
	if err != nil {
		t.Errorf("could not retreive unused projects, error: %v", err)
	}
	if len(remainingProjects) != 1 {
		t.Errorf("expected 1 remaining projects, but got %d", len(remainingProjects))
	}
}
