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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/mock/gomock"
	"google.golang.org/api/iterator"
)

func TestRoundRobinQueueTasksNoProjects(t *testing.T) {
	// Test that fallback logic works when no projects are returned.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockProcessListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockProjectsRowIterator := gcloud.NewMockRowIterator(mockCtrl)
	mockReadTasksTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockTasksRowIterator := gcloud.NewMockRowIterator(mockCtrl)

	tc := &SpannerStore{mockSpanner, mockPubSub}

	mockProjectsRowIterator.EXPECT().Do(gomock.Any()).Return(nil)
	mockProjectsRowIterator.EXPECT().Stop()
	mockTasksRowIterator.EXPECT().Next().Return(nil, iterator.Done)
	mockTasksRowIterator.EXPECT().Stop()

	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mockProjectsRowIterator)
	mockReadTasksTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mockTasksRowIterator)

	spannerSingleCallNumber := 0
	mockSpanner.EXPECT().Single().DoAndReturn(func() *gcloud.MockReadOnlyTransaction {
		if spannerSingleCallNumber == 0 {
			spannerSingleCallNumber++
			return mockReadProjectsTransaction
		} else {
			return mockReadTasksTransaction
		}
	}).MaxTimes(2)

	mockListTopic.EXPECT().Stop()
	mockCopyTopic.EXPECT().Stop()
	mockPubSub.EXPECT().TopicInProject("cloud-ingest-list", "fakeProjectID").Return(mockListTopic)
	mockPubSub.EXPECT().TopicInProject("cloud-ingest-copy", "fakeProjectID").Return(mockCopyTopic)

	tc.RoundRobinQueueTasks(1, mockProcessListTopic, "fakeProjectID")
}
