/*
Copyright 2019 Google Inc. All Rights Reserved.
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

package delete

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/googleapi"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func TestDeleteBundle(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type bundledObjectTestData struct {
		size           int64
		bucket         string
		objectName     string
		genNum         int64
		status         taskpb.Status
		failureType    taskpb.FailureType
		failureMessage string
		log            *taskpb.BundledObjectLog

		wantStatus         taskpb.Status
		wantFailureType    taskpb.FailureType
		wantFailureMessage string
		wantRetryTimes     int
		wantError          error
	}

	object_not_found_error := &googleapi.Error{Code: http.StatusNotFound, Message: "object not found"}
	gateway_error := &googleapi.Error{Code: http.StatusBadGateway, Message: "bad gateway"}
	permission_denied_error := &googleapi.Error{Code: http.StatusUnauthorized, Message: "permission denied"}

	tests := []struct {
		desc           string
		bundledObjects []*bundledObjectTestData
		bundleStatus   taskpb.Status
		bundleFailure  taskpb.FailureType
		bundleLog      *taskpb.DeleteBundleLog
	}{
		{
			desc: "test delete bundle success",
			bundledObjects: []*bundledObjectTestData{
				&bundledObjectTestData{
					size:           19,
					bucket:         "bucket",
					objectName:     "object1",
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         1,
					wantError:      nil,
					wantRetryTimes: 1,
				},
				&bundledObjectTestData{
					size:           18,
					bucket:         "bucket",
					objectName:     "object2",
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         2,
					wantError:      nil,
					wantRetryTimes: 1,
				},
			},
			bundleStatus: taskpb.Status_SUCCESS,
			bundleLog: &taskpb.DeleteBundleLog{
				ObjectsDeleted: 2,
				BytesDeleted:   37,
			},
		},
		{
			desc: "test delete bundle success with not found",
			bundledObjects: []*bundledObjectTestData{
				&bundledObjectTestData{
					size:           19,
					bucket:         "bucket",
					objectName:     "object1",
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         1,
					wantError:      nil,
					wantRetryTimes: 1,
				},
				&bundledObjectTestData{
					size:           18,
					bucket:         "bucket",
					objectName:     "object2",
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         2,
					wantError:      object_not_found_error,
					wantRetryTimes: 1,
				},
			},
			bundleStatus: taskpb.Status_SUCCESS,
			bundleLog: &taskpb.DeleteBundleLog{
				ObjectsDeleted: 2,
				BytesDeleted:   37,
				ObjectsFailed:  0,
				BytesFailed:    0,
			},
		},
		{
			desc: "test delete bundle partial failure",
			bundledObjects: []*bundledObjectTestData{
				&bundledObjectTestData{
					size:           19,
					bucket:         "bucket",
					objectName:     "object1",
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         1,
					wantError:      nil,
					wantRetryTimes: 1,
				},
				&bundledObjectTestData{
					size:               18,
					bucket:             "bucket",
					objectName:         "object2",
					wantStatus:         taskpb.Status_FAILED,
					genNum:             2,
					wantError:          permission_denied_error,
					wantFailureType:    taskpb.FailureType_PERMISSION_FAILURE,
					wantFailureMessage: fmt.Sprint(permission_denied_error),
					wantRetryTimes:     1,
				},
			},
			bundleStatus:  taskpb.Status_FAILED,
			bundleFailure: taskpb.FailureType_UNKNOWN_FAILURE,
			bundleLog: &taskpb.DeleteBundleLog{
				ObjectsDeleted: 1,
				BytesDeleted:   19,
				ObjectsFailed:  1,
				BytesFailed:    18,
			},
		},
		{
			desc: "test delete bundle failure",
			bundledObjects: []*bundledObjectTestData{
				&bundledObjectTestData{
					size:               19,
					bucket:             "bucket",
					objectName:         "object1",
					wantStatus:         taskpb.Status_FAILED,
					genNum:             1,
					wantError:          gateway_error,
					wantFailureType:    taskpb.FailureType_UNKNOWN_FAILURE,
					wantFailureMessage: fmt.Sprint(gateway_error),
					wantRetryTimes:     maxRetryCount,
				},
				&bundledObjectTestData{
					size:               18,
					bucket:             "bucket",
					objectName:         "object2",
					wantStatus:         taskpb.Status_FAILED,
					genNum:             2,
					wantError:          permission_denied_error,
					wantFailureType:    taskpb.FailureType_PERMISSION_FAILURE,
					wantFailureMessage: fmt.Sprint(permission_denied_error),
					wantRetryTimes:     1,
				},
			},
			bundleStatus:  taskpb.Status_FAILED,
			bundleFailure: taskpb.FailureType_UNKNOWN_FAILURE,
			bundleLog: &taskpb.DeleteBundleLog{
				ObjectsFailed: 2,
				BytesFailed:   37,
			},
		},
		{
			desc: "test delete bundle retry with a permanent failure and success",
			bundledObjects: []*bundledObjectTestData{
				&bundledObjectTestData{
					size:               19,
					bucket:             "bucket",
					objectName:         "object1",
					wantStatus:         taskpb.Status_FAILED,
					genNum:             1,
					wantError:          gateway_error,
					wantFailureType:    taskpb.FailureType_UNKNOWN_FAILURE,
					wantFailureMessage: fmt.Sprint(gateway_error),
					wantRetryTimes:     maxRetryCount,
				},
				&bundledObjectTestData{
					size:           18,
					bucket:         "bucket",
					objectName:     "object2",
					status:         taskpb.Status_FAILED,
					failureType:    taskpb.FailureType_PERMISSION_FAILURE,
					failureMessage: fmt.Sprint(permission_denied_error),
					log: &taskpb.BundledObjectLog{
						DstBucket:      "bucket",
						DstObject:      "object2",
						DstObjectBytes: 18,
						Status:         taskpb.Status_FAILED,
						FailureMessage: fmt.Sprint(permission_denied_error),
						FailureType:    taskpb.FailureType_PERMISSION_FAILURE,
					},
					wantStatus:         taskpb.Status_FAILED,
					genNum:             2,
					wantFailureType:    taskpb.FailureType_PERMISSION_FAILURE,
					wantFailureMessage: fmt.Sprint(permission_denied_error),
					wantRetryTimes:     0,
				},
				&bundledObjectTestData{
					size:       19,
					bucket:     "bucket",
					objectName: "object3",
					status:     taskpb.Status_SUCCESS,
					log: &taskpb.BundledObjectLog{
						DstBucket:      "bucket",
						DstObject:      "object3",
						DstObjectBytes: 19,
						Status:         taskpb.Status_SUCCESS,
					},
					wantStatus:     taskpb.Status_SUCCESS,
					genNum:         1,
					wantRetryTimes: 0,
				},
			},
			bundleStatus:  taskpb.Status_FAILED,
			bundleFailure: taskpb.FailureType_UNKNOWN_FAILURE,
			bundleLog: &taskpb.DeleteBundleLog{
				ObjectsDeleted: 1,
				BytesDeleted:   19,
				ObjectsFailed:  2,
				BytesFailed:    37,
			},
		},
	}
	for _, tc := range tests {
		mockGCS := gcloud.NewMockGCS(mockCtrl)
		bundleSpec := &taskpb.DeleteBundleSpec{}

		for _, object := range tc.bundledObjects {
			bundleSpec.BundledObjects = append(bundleSpec.BundledObjects, &taskpb.BundledObject{
				DeleteObjectSpec: &taskpb.DeleteObjectSpec{
					DstBucket:      object.bucket,
					DstObject:      object.objectName,
					DstObjectBytes: object.size,
					GenerationNum:  object.genNum,
				},
				Status:           object.status,
				FailureType:      object.failureType,
				FailureMessage:   object.failureMessage,
				BundledObjectLog: object.log,
			})

			mockGCS.EXPECT().DeleteObject(
				context.Background(), object.bucket, object.objectName, object.genNum).Return(object.wantError).Times(object.wantRetryTimes)
		}

		h := DeleteHandler{
			gcs:                 mockGCS,
			concurrentDeleteSem: semaphore.NewWeighted(1),
		}
		taskReqMsg := &taskpb.TaskReqMsg{
			TaskRelRsrcName: "task",
			Spec:            &taskpb.Spec{Spec: &taskpb.Spec_DeleteBundleSpec{bundleSpec}},
		}
		taskRespMsg := h.Do(context.Background(), taskReqMsg)

		// Check for the overall task status.
		t.Logf("DeleteHandler.Do(%q)", tc.desc)
		if tc.bundleStatus == taskpb.Status_SUCCESS {
			if isValid, errMsg := common.IsValidSuccessMsg("task", taskRespMsg); !isValid {
				t.Error(errMsg)
			}
		} else {
			if isValid, errMsg := common.IsValidFailureMsg("task", tc.bundleFailure, taskRespMsg); !isValid {
				t.Error(errMsg)
			}
		}

		// Check the status of each of the deleted objects.
		resBundledObjects := taskRespMsg.RespSpec.GetDeleteBundleSpec().BundledObjects
		for i, object := range tc.bundledObjects {
			wantLog := &taskpb.BundledObjectLog{
				DstBucket:      object.bucket,
				DstObject:      object.objectName,
				DstObjectBytes: object.size,
				Status:         object.wantStatus,
				FailureMessage: object.wantFailureMessage,
				FailureType:    object.wantFailureType,
			}

			if resBundledObjects[i].Status != object.wantStatus {
				t.Errorf("DeleteHandler.Do(%q), got status: %s, want: %s", tc.desc, resBundledObjects[i].Status, object.wantStatus)
			}

			if !proto.Equal(resBundledObjects[i].BundledObjectLog, wantLog) {
				t.Errorf("DeleteHandler.Do(%q), file %d log = %+v, want: %+v", tc.desc, i, resBundledObjects[i].BundledObjectLog, wantLog)
			}

			if resBundledObjects[i].FailureType != object.wantFailureType {
				t.Errorf("DeleteHandler.Do(%q), got failureType: %s, want: %s", tc.desc, resBundledObjects[i].FailureType, object.wantFailureType)
			}

			tc.bundleLog.BundledObjectsLogs = append(tc.bundleLog.BundledObjectsLogs, wantLog)
		}

		// Check for the overall bundle log.
		wantLog := &taskpb.Log{Log: &taskpb.Log_DeleteBundleLog{tc.bundleLog}}
		if !proto.Equal(taskRespMsg.Log, wantLog) {
			t.Errorf("DeleteHandler.Do(%q), got log = %+v, want: %+v", tc.desc, taskRespMsg.Log, wantLog)
		}
	}
}
