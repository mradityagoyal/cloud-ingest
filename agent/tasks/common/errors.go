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

package common

import (
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const MaxRetryCount = 1

type AgentError struct {
	Msg         string
	FailureType taskpb.FailureType
}

func (ae AgentError) Error() string {
	return ae.Msg
}

// IsRetryableError returns true if an error is retryable and false otherwise.
func IsRetryableError(err error) bool {
	switch GetFailureTypeFromError(err) {
	case taskpb.FailureType_PERMISSION_FAILURE, taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskpb.FailureType_SOURCE_DIR_NOT_FOUND, taskpb.FailureType_PRECONDITION_FAILURE:
		return false
	default:
		return true
	}
}
