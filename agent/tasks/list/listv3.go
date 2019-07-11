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
package list

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// ListHandlerV3 is responsible for handling job run version 3 depth-first list tasks.
// When this handler processes a list task, it produces two files. The first file is a list file
// that contains all the contents of a listed directory (both files and child directories are listed).
// The second file contains a list of all the unexplored directories (directories that were
// discovered but not yet listed).
// This style of listing allows us to easily detect the deletion of directories.
type ListHandlerV3 struct {
	gcs                   gcloud.GCS
	resumableChunkSize    int
	listFileSizeThreshold int
	allowedDirBytes       int
	statsTracker          *stats.Tracker // For tracking bytes sent/copied.
}

// NewListHandlerV3 returns a new ListHandlerV3.
func NewListHandlerV3(storageClient *storage.Client, st *stats.Tracker) *ListHandlerV3 {
	// Convert maxMemoryForListingDirectories to bytes and divide it equally between
	// the list task processing threads.
	allowedDirBytes := *maxMemoryForListingDirectories * 1024 * 1024 / *NumberConcurrentListTasks
	return &ListHandlerV3{
		gcs:                   gcloud.NewGCSClient(storageClient),
		resumableChunkSize:    *listTaskChunkSize,
		listFileSizeThreshold: *listFileSizeThreshold,
		allowedDirBytes:       allowedDirBytes,
		statsTracker:          st,
	}
}

func (h *ListHandlerV3) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	listSpec := taskReqMsg.Spec.GetListSpec()
	if listSpec == nil {
		err := errors.New("ListHandlerV3.Do taskReqMsg.Spec is not ListSpec")
		return common.BuildTaskRespMsg(taskReqMsg, nil, nil, err)
	}

	log := &taskpb.Log{
		Log: &taskpb.Log_ListLog{ListLog: &taskpb.ListLog{}},
	}

	// Write list file BEFORE the unexplored dirs file. This ordering is important to ensure that if
	// two agents are processing the same task, one will succeed and the other will fail.
	listFileW := gcsWriterWithCondition(ctx, h.gcs, listSpec.DstListResultBucket, listSpec.DstListResultObject, listSpec.ListResultExpectedGenerationNum, h.resumableChunkSize)
	listBtw := h.statsTracker.NewListByteTrackingWriter(listFileW, true)

	settings := listSettings{
		listFileSizeThreshold: h.listFileSizeThreshold,
		maxDirBytes:           h.allowedDirBytes,
		includeDirs:           true,
		includeDirHeader:      true,
	}
	listMD, unlistedDirs, err := listDirectoriesAndWriteResults(listBtw, listSpec, settings, h.statsTracker)
	if err != nil {
		listFileW.CloseWithError(err)
		if os.IsNotExist(err) {
			err = common.AgentError{
				FailureType: taskpb.FailureType_SOURCE_DIR_NOT_FOUND,
				Msg:         fmt.Sprintf("Could not find the job's source directory (%q). Error: %q", listSpec.RootDirectory, err.Error()),
			}
		}
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}
	if err := listFileW.Close(); err != nil {
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	unexploredDirsW := gcsWriterWithCondition(ctx, h.gcs, listSpec.DstListResultBucket, listSpec.DstUnexploredDirsObject, listSpec.UnexploredDirsExpectedGenerationNum, h.resumableChunkSize)
	unexploredBtw := h.statsTracker.NewListByteTrackingWriter(unexploredDirsW, false)
	if err = writeDirectories(unexploredBtw, unlistedDirs); err != nil {
		unexploredDirsW.CloseWithError(err)
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	if err := unexploredDirsW.Close(); err != nil {
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	setListLog(log, listMD)

	return common.BuildTaskRespMsg(taskReqMsg, nil, log, nil)
}
