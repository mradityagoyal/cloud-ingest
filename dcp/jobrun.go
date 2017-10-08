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

const (
	JobNotStarted int64 = 0
	JobInProgress int64 = 1
	JobFailed     int64 = 2
	JobSuccess    int64 = 3
)

type ListTaskProgressSpec struct {
	Status int `json:"status"`
}

type UploadGCSTaskProgressSpec struct {
	Status         int64 `json:"status"`
	TotalFiles     int64 `json:"totalFiles"`
	FilesCompleted int64 `json:"filesCompleted"`
	FilesFailed    int64 `json:"filesFailed"`
}

type LoadBQTaskProgressSpec struct {
	Status           int64 `json:"status"`
	TotalObjects     int64 `json:"totalObjects"`
	ObjectsCompleted int64 `json:"objectsCompleted"`
	ObjectsFailed    int64 `json:"objectsFailed"`
}

type JobProgressSpec struct {
	TotalTasks     int64 `json:"totalTasks"`
	TasksCompleted int64 `json:"tasksCompleted"`
	TasksFailed    int64 `json:"tasksFailed"`
	// Store the progress of each task type as a pointer so it's
	// omitted when empty
	ListProgress      *ListTaskProgressSpec      `json:"list,omitempty"`
	UploadGCSProgress *UploadGCSTaskProgressSpec `json:"uploadGCS,omitempty"`
	LoadBQProgress    *LoadBQTaskProgressSpec    `json:"loadBigQuery,omitempty"`
}

type JobProgressDelta struct {
	NewTasks       int64
	TasksCompleted int64
	TasksFailed    int64
}

type JobRun struct {
	JobConfigId     string
	JobRunId        string
	JobCreationTime int64
	Status          int64
	Progress        string
}

type JobRunFullId struct {
	JobConfigId string
	JobRunId    string
}

// ApplyDelta applies the changes in the given deltaObj to this
func (j *JobProgressSpec) ApplyDelta(deltaObj *JobProgressDelta) {
	j.TotalTasks += deltaObj.NewTasks
	j.TasksCompleted += deltaObj.TasksCompleted
	j.TasksFailed += deltaObj.TasksFailed
}

// GetJobStatus returns the status of the job with this ProgressSpec
func (j *JobProgressSpec) GetJobStatus() int64 {
	var status int64
	if j.TotalTasks == 0 {
		status = JobNotStarted
	} else if j.TotalTasks == j.TasksCompleted {
		status = JobSuccess
	} else if j.TotalTasks == (j.TasksCompleted + j.TasksFailed) {
		status = JobFailed
	} else {
		status = JobInProgress
	}
	return status
}

// IsJobTerminated returns whether a job has terminated or not.
func IsJobTerminated(jobStatus int64) bool {
	return jobStatus == JobFailed || jobStatus == JobSuccess
}
