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
)

func TestGetJobStatusNotStarted(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     0,
		TasksCompleted: 0,
		TasksFailed:    0,
	}
	status := progressObj.GetJobStatus()
	if status != JobNotStarted {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobNotStarted, status)
	}
}

func TestGetJobStatusInProgressNoFailures(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     5,
		TasksCompleted: 3,
		TasksFailed:    0,
	}
	status := progressObj.GetJobStatus()
	if status != JobInProgress {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobInProgress, status)
	}
}

func TestGetJobStatusInProgressWithFailures(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     5,
		TasksCompleted: 3,
		TasksFailed:    1,
	}
	status := progressObj.GetJobStatus()
	if status != JobInProgress {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobInProgress, status)
	}
}

func TestGetJobStatusSuccess(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     5,
		TasksCompleted: 5,
		TasksFailed:    0,
	}
	status := progressObj.GetJobStatus()
	if status != JobSuccess {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobSuccess, status)
	}
}

func TestGetJobStatusFailureMixture(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     5,
		TasksCompleted: 4,
		TasksFailed:    1,
	}
	status := progressObj.GetJobStatus()
	if status != JobFailed {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobFailed, status)
	}
}

func TestGetJobStatusFailureAllFails(t *testing.T) {
	progressObj := JobProgressSpec{
		TotalTasks:     5,
		TasksCompleted: 0,
		TasksFailed:    5,
	}
	status := progressObj.GetJobStatus()
	if status != JobFailed {
		t.Errorf("expected job status for %+v to be %d, instead found %d",
			progressObj, JobFailed, status)
	}
}
