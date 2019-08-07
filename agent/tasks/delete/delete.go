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
	"errors"
	"flag"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/sync/semaphore"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

var (
	concurrentDeleteMax = flag.Int("concurrent-delete-max", 10, "The maximum allowed number of concurrent delete objects.")
)

const maxRetryCount = 3

// DeleteHandler is responsible for handling delete tasks.
type DeleteHandler struct {
	gcs                 gcloud.GCS
	concurrentDeleteSem *semaphore.Weighted // Limits the number of concurrent goroutines deleting objects.
	statsTracker        *stats.Tracker      // For tracking bytes deleted.
}

// isRetryableError returns true if an error is retryable and false otherwise.
func isRetryableError(err error) bool {
	switch common.GetFailureTypeFromError(err) {
	case taskpb.FailureType_PERMISSION_FAILURE, taskpb.FailureType_PRECONDITION_FAILURE:
		return false
	default:
		return true
	}
}

// NewDeleteHandler creates a DeleteHandler with storage.Client and http.Client.
func NewDeleteHandler(storageClient *storage.Client, st *stats.Tracker) *DeleteHandler {
	return &DeleteHandler{
		gcs:                 gcloud.NewGCSClient(storageClient),
		concurrentDeleteSem: semaphore.NewWeighted(int64(*concurrentDeleteMax)),
		statsTracker:        st,
	}
}

// Do implements a handler to delete a bundle of objects in GCS.
func (h *DeleteHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg, _ time.Time) *taskpb.TaskRespMsg {
	var respSpec *taskpb.Spec
	var log *taskpb.Log
	var err error

	if taskReqMsg.Spec.GetDeleteBundleSpec() != nil {
		var dbl *taskpb.DeleteBundleLog
		bundleSpec := proto.Clone(taskReqMsg.Spec.GetDeleteBundleSpec()).(*taskpb.DeleteBundleSpec)
		dbl, err = h.handleDeleteBundleSpec(ctx, bundleSpec)
		respSpec = &taskpb.Spec{Spec: &taskpb.Spec_DeleteBundleSpec{bundleSpec}}
		log = &taskpb.Log{Log: &taskpb.Log_DeleteBundleLog{dbl}}
	} else {
		err = errors.New("DeleteHandler.Do taskReqMsg.Spec is not DeleteBundleSpec")
	}

	return common.BuildTaskRespMsg(taskReqMsg, respSpec, log, err)
}

func (h *DeleteHandler) handleDeleteBundleSpec(ctx context.Context, bundleSpec *taskpb.DeleteBundleSpec) (*taskpb.DeleteBundleLog, error) {
	var wg sync.WaitGroup
	for _, bo := range bundleSpec.BundledObjects {
		// In case of end to end retries we do not want to retry successes and permanent failures.
		if bo.Status != taskpb.Status_SUCCESS && bo.Status != taskpb.Status_FAILED {
			wg.Add(1)
			go func(bo *taskpb.BundledObject) {
				defer wg.Done()
				bo.BundledObjectLog = h.handleDeleteObjectSpec(ctx, bo.DeleteObjectSpec)
				bo.FailureType = bo.BundledObjectLog.FailureType
				bo.FailureMessage = bo.BundledObjectLog.FailureMessage
				bo.Status = bo.BundledObjectLog.Status
			}(bo)
		}
	}
	wg.Wait()
	return getBundleLogAndError(bundleSpec)
}

func getBundleLogAndError(bs *taskpb.DeleteBundleSpec) (*taskpb.DeleteBundleLog, error) {
	var log taskpb.DeleteBundleLog
	for _, bo := range bs.BundledObjects {
		if bo.Status == taskpb.Status_SUCCESS {
			log.ObjectsDeleted++
			log.BytesDeleted += bo.BundledObjectLog.DstObjectBytes
		} else {
			log.ObjectsFailed++
			log.BytesFailed += bo.BundledObjectLog.DstObjectBytes
			glog.Warningf("bundledObject %v from bucket %v, failed with err: %v", bo.DeleteObjectSpec.DstObject, bo.DeleteObjectSpec.DstBucket, bo.FailureMessage)
		}
		log.BundledObjectsLogs = append(log.BundledObjectsLogs, bo.BundledObjectLog)
	}
	var err error
	if log.ObjectsFailed > 0 {
		err = common.AgentError{
			Msg:         fmt.Sprintf("DeleteBundle had %v failures", log.ObjectsFailed),
			FailureType: taskpb.FailureType_UNKNOWN_FAILURE,
		}
	}
	return &log, err
}

func (h *DeleteHandler) handleDeleteObjectSpec(ctx context.Context, deleteSpec *taskpb.DeleteObjectSpec) *taskpb.BundledObjectLog {
	h.concurrentDeleteSem.Acquire(ctx, 1)
	defer h.concurrentDeleteSem.Release(1)

	var err error
	for i := 0; i < maxRetryCount; i++ {
		err = h.gcs.DeleteObject(ctx, deleteSpec.DstBucket, deleteSpec.DstObject, deleteSpec.GenerationNum)
		// Reset object not found failure to appear like there was no failure.
		// If an object was not found in the destination bucket at the time of deletion, the resultant
		// state is the same as an object being deleted successfully.
		if err == storage.ErrObjectNotExist {
			err = nil
		}
		if err == nil || !isRetryableError(err) {
			break
		}
		h.statsTracker.RecordPulseStats(&stats.PulseStats{DeleteInternalRetries: 1})
	}

	dl := &taskpb.BundledObjectLog{
		DstBucket:      deleteSpec.DstBucket,
		DstObject:      deleteSpec.DstObject,
		DstObjectBytes: deleteSpec.DstObjectBytes,
	}

	if err != nil {
		dl.FailureType = common.GetFailureTypeFromError(err)
		dl.FailureMessage = err.Error()
		dl.Status = taskpb.Status_FAILED
	} else {
		dl.Status = taskpb.Status_SUCCESS
	}

	return dl
}
