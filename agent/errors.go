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

package agent

import (
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
)

type AgentError struct {
	Msg         string
	FailureType proto.TaskFailureType_Type
}

func (ae AgentError) Error() string {
	return ae.Msg
}

func NewInvalidTaskParamsError(taskParams taskParams, err error) *AgentError {
	return &AgentError{
		Msg:         fmt.Sprintf("Invalid task params arguments: %+v, err: %v", taskParams, err),
		FailureType: proto.TaskFailureType_UNKNOWN,
	}
}
