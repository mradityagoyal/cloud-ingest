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
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/mock/gomock"
	"google.golang.org/api/iterator"
)

func TestRoundRobinQueueTasksNoProjects(t *testing.T) {
	// Tests that RoundRobinQueueTasks does nothing if there are no projects.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockProcessListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockProjectsRowIterator := gcloud.NewMockRowIterator(mockCtrl)

	tc := &SpannerStore{mockSpanner, mockPubSub}

	mockProjectsRowIterator.EXPECT().Do(gomock.Any()).Return(nil)
	mockProjectsRowIterator.EXPECT().Stop()
	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mockProjectsRowIterator)
	mockSpanner.EXPECT().Single().Return(mockReadProjectsTransaction).MaxTimes(1)

	tc.RoundRobinQueueTasks(1, mockProcessListTopic)
}

func TestRoundRobinQueueTasksTwoProjectsNoTasks(t *testing.T) {
	// Test that when projects are populated in Spanner, appropriate calls are made to PubSub.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockProcessListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockReadTasksTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockTasksRowIterator := gcloud.NewMockRowIterator(mockCtrl)

	tc := &SpannerStore{mockSpanner, mockPubSub}

	columnNames := []string{"ProjectId", "ListTopicId", "CopyTopicId"}
	row1, _ := spanner.NewRow(columnNames, []interface{}{"fakeProjectID1", "lt1", "ct1"})
	row2, _ := spanner.NewRow(columnNames, []interface{}{"fakeProjectID2", "lt2", "ct2"})
	projectsRowIterator := gcloud.NewFakeRowIterator([]spanner.Row{*row1, *row2})
	// Projects should be queried once.
	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(projectsRowIterator)

	// Tasks should be queried twice, once for each project.
	mockTasksRowIterator.EXPECT().Next().MaxTimes(2).Return(nil, iterator.Done)
	mockTasksRowIterator.EXPECT().Stop().MaxTimes(2)
	mockReadTasksTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).MaxTimes(2).Return(mockTasksRowIterator)

	spannerSingleCallNumber := 0
	mockSpanner.EXPECT().Single().DoAndReturn(func() *gcloud.MockReadOnlyTransaction {
		spannerSingleCallNumber++
		if spannerSingleCallNumber == 1 {
			return mockReadProjectsTransaction
		} else {
			return mockReadTasksTransaction
		}
	}).MaxTimes(3)

	mockListTopic.EXPECT().Stop().MaxTimes(2)
	mockCopyTopic.EXPECT().Stop().MaxTimes(2)
	mockPubSub.EXPECT().TopicInProject("lt1", "fakeProjectID1").Return(mockListTopic)
	mockPubSub.EXPECT().TopicInProject("lt2", "fakeProjectID2").Return(mockListTopic)
	mockPubSub.EXPECT().TopicInProject("ct1", "fakeProjectID1").Return(mockCopyTopic)
	mockPubSub.EXPECT().TopicInProject("ct2", "fakeProjectID2").Return(mockCopyTopic)

	tc.RoundRobinQueueTasks(3, mockProcessListTopic)
}

func TestRoundRobinQueueTasksTwoProjectsAndTasks(t *testing.T) {
	// Test that when projects are populated in Spanner, appropriate calls are made to PubSub.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockPubSub := gcloud.NewMockPS(mockCtrl)
	mockListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockProcessListTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockCopyTopic := gcloud.NewMockPSTopic(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockReadTasksTransaction1 := gcloud.NewMockReadOnlyTransaction(mockCtrl)
	mockReadTasksTransaction2 := gcloud.NewMockReadOnlyTransaction(mockCtrl)

	tc := &SpannerStore{mockSpanner, mockPubSub}

	projectsColumnNames := []string{"ProjectId", "ListTopicId", "CopyTopicId"}
	projectsRow1, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID1", "lt1", "ct1"})
	projectsRow2, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID2", "lt2", "ct2"})
	projectsRowIterator := gcloud.NewFakeRowIterator([]spanner.Row{*projectsRow1, *projectsRow2})

	tasksColumnNames := []string{"ProjectId", "JobConfigId", "JobRunId", "TaskId", "TaskType", "TaskSpec"}
	tasksIter1Row1, _ := spanner.NewRow(
		tasksColumnNames, []interface{}{
			"fakeProjectID1",
			"jc1",
			"jr1",
			"copy1",
			copyTaskType,
			`{
          "src_file": "file1",
          "dst_bucket": "bucket1",
          "dst_object": "file1",
          "expected_generation_num": "0"
      }`,
		})
	tasksIter1Row2, _ := spanner.NewRow(
		tasksColumnNames, []interface{}{
			"fakeProjectID1",
			"jc1",
			"jr1",
			"copy2",
			copyTaskType,
			`{
          "src_file": "file2",
          "dst_bucket": "bucket1",
          "dst_object": "file2",
          "expected_generation_num": "0"
      }`,
		})
	tasksIter2Row1, _ := spanner.NewRow(
		tasksColumnNames, []interface{}{
			"fakeProjectID2",
			"jc1",
			"jr1",
			"copy1",
			copyTaskType,
			`{
          "src_file": "file1",
          "dst_bucket": "bucket2",
          "dst_object": "file1",
          "expected_generation_num": "0"
      }`,
		})
	mockTasksRowIterator1 := gcloud.NewFakeRowIterator([]spanner.Row{*tasksIter1Row1, *tasksIter1Row2})
	mockTasksRowIterator2 := gcloud.NewFakeRowIterator([]spanner.Row{*tasksIter2Row1})

	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(projectsRowIterator)
	mockReadTasksTransaction1.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mockTasksRowIterator1)
	mockReadTasksTransaction2.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mockTasksRowIterator2)

	spannerSingleCallNumber := 0
	mockSpanner.EXPECT().Single().DoAndReturn(func() *gcloud.MockReadOnlyTransaction {
		spannerSingleCallNumber++
		if spannerSingleCallNumber == 1 {
			return mockReadProjectsTransaction
		} else if spannerSingleCallNumber == 2 {
			return mockReadTasksTransaction1
		} else {
			return mockReadTasksTransaction2
		}
	}).MaxTimes(3)

	// Theoretically, Publish should only be called twice because round-robin
	// queuing is configured below to retrieve at most one task for each of the two projects,
	// but the fake Spanner implementation is not that sophisticated (max tasks is handled in the
	// query itself.)
	mockPublishResult := gcloud.NewMockPSPublishResult(mockCtrl)
	mockPublishResult.EXPECT().Get(gomock.Any()).MaxTimes(3).Return("fakeServerID", nil)
	mockCopyTopic.EXPECT().Publish(gomock.Any(), gomock.Any()).MaxTimes(3).Return(mockPublishResult)
	mockListTopic.EXPECT().Stop().MaxTimes(2)
	mockCopyTopic.EXPECT().Stop().MaxTimes(2)

	mockPubSub.EXPECT().TopicInProject("lt1", "fakeProjectID1").Return(mockListTopic)
	mockPubSub.EXPECT().TopicInProject("ct1", "fakeProjectID1").Return(mockCopyTopic)
	mockPubSub.EXPECT().TopicInProject("lt2", "fakeProjectID2").Return(mockListTopic)
	mockPubSub.EXPECT().TopicInProject("ct2", "fakeProjectID2").Return(mockCopyTopic)
	mockSpanner.EXPECT().ReadWriteTransaction(gomock.Any(), gomock.Any()).MaxTimes(2).Return(time.Unix(0, 0), nil)

	tc.RoundRobinQueueTasks(1, mockProcessListTopic)
}

func TestGetUnusedProjectsReturnMax(t *testing.T) {
	// Tests that GetUnusedProjects correctly parses Spanner rows and respects return limits.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)

	tc := &SpannerStore{mockSpanner, nil}

	projectsColumnNames := []string{
		"ProjectId", "ListTopicId", "CopyTopicId", "ListProgressSubscriptionId",
		"CopyProgressSubscriptionId"}
	projectsRow1, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID1", "lt1", "ct1", "ls1", "cs1"})
	projectsRow2, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID2", "lt2", "ct2", "ls2", "cs2"})
	projectsRowIterator := gcloud.NewFakeRowIterator([]spanner.Row{*projectsRow1, *projectsRow2})

	mockSpanner.EXPECT().Single().Return(mockReadProjectsTransaction).MaxTimes(1)
	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(projectsRowIterator).MaxTimes(1)

	projectInfos, err := tc.GetUnusedProjects(2)
	if err != nil {
		t.Errorf("GetUnusedProjects returned error: %v", err)
	} else if len(projectInfos) != 2 {
		t.Errorf("Expected 2 projectInfos, got %d", len(projectInfos))
	} else {
		DeepEqualCompare("GetUnusedProjects",
			ProjectInfo{
				ProjectID:                  "fakeProjectID1",
				ListTopicID:                "lt1",
				CopyTopicID:                "ct1",
				ListProgressSubscriptionID: "ls1",
				CopyProgressSubscriptionID: "cs1",
			},
			*projectInfos[0],
			t)
		DeepEqualCompare("GetUnusedProjects",
			ProjectInfo{
				ProjectID:                  "fakeProjectID2",
				ListTopicID:                "lt2",
				CopyTopicID:                "ct2",
				ListProgressSubscriptionID: "ls2",
				CopyProgressSubscriptionID: "cs2",
			},
			*projectInfos[1],
			t)
	}
}

func TestGetUnusedProjectsReturnFewer(t *testing.T) {
	// Tests that GetUnusedProjects correctly parses Spanner rows and respects return limits.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSpanner := gcloud.NewMockSpanner(mockCtrl)
	mockReadProjectsTransaction := gcloud.NewMockReadOnlyTransaction(mockCtrl)

	tc := &SpannerStore{mockSpanner, nil}

	projectsColumnNames := []string{
		"ProjectId", "ListTopicId", "CopyTopicId", "ListProgressSubscriptionId",
		"CopyProgressSubscriptionId"}
	projectsRow1, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID1", "lt1", "ct1", "ls1", "cs1"})
	projectsRow2, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID2", "lt2", "ct2", "ls2", "cs2"})
	projectsRow3, _ := spanner.NewRow(
		projectsColumnNames, []interface{}{"fakeProjectID3", "lt3", "ct3", "ls3", "cs3"})
	projectsRowIterator := gcloud.NewFakeRowIterator([]spanner.Row{
		*projectsRow1, *projectsRow2, *projectsRow3})

	mockSpanner.EXPECT().Single().Return(mockReadProjectsTransaction).MaxTimes(1)
	mockReadProjectsTransaction.EXPECT().Query(gomock.Any(), gomock.Any()).Return(projectsRowIterator).MaxTimes(1)

	projectInfos, err := tc.GetUnusedProjects(5)
	if err != nil {
		t.Errorf("GetUnusedProjects returned error: %v", err)
	}
	if len(projectInfos) != 3 {
		t.Errorf("Expected 3 projectInfos, got %d", len(projectInfos))
	}
	DeepEqualCompare("GetUnusedProjects",
		ProjectInfo{
			ProjectID:                  "fakeProjectID1",
			ListTopicID:                "lt1",
			CopyTopicID:                "ct1",
			ListProgressSubscriptionID: "ls1",
			CopyProgressSubscriptionID: "cs1",
		},
		*projectInfos[0],
		t)
	DeepEqualCompare("GetUnusedProjects",
		ProjectInfo{
			ProjectID:                  "fakeProjectID2",
			ListTopicID:                "lt2",
			CopyTopicID:                "ct2",
			ListProgressSubscriptionID: "ls2",
			CopyProgressSubscriptionID: "cs2",
		},
		*projectInfos[1],
		t)
	DeepEqualCompare("GetUnusedProjects",
		ProjectInfo{
			ProjectID:                  "fakeProjectID3",
			ListTopicID:                "lt3",
			CopyTopicID:                "ct3",
			ListProgressSubscriptionID: "ls3",
			CopyProgressSubscriptionID: "cs3",
		},
		*projectInfos[2],
		t)

}
