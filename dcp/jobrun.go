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
	"encoding/json"
	"errors"
	"fmt"
)

const (
	JobNotStarted int64 = 0
	JobInProgress int64 = 1
	JobFailed     int64 = 2
	JobSuccess    int64 = 3

	// Keys for the task status counters.
	KeyTotalTasks     string = "totalTasks"
	KeyTasksCompleted string = "tasksCompleted"
	KeyTasksFailed    string = "tasksFailed"
	KeyTasksQueued    string = "tasksQueued"

	KeySuffixList string = "List"
	KeySuffixCopy string = "Copy"
	KeySuffixLoad string = "Load"

	// Keys for the log entry counters.
	KeyListFilesFound     string = "listFilesFound"
	KeyListBytesFound     string = "listBytesFound"
	KeyListFileStatErrors string = "listFileStatErrors"
	KeyBytesCopied        string = "bytesCopied"
)

// The counter field is encoded directly into the JSON, not the entire struct.
type JobCounters struct {
	counter map[string]int64
}

type JobCountersCollection struct {
	deltas map[JobRunFullId]*JobCounters
}

type JobRun struct {
	JobConfigId     string
	JobRunId        string
	JobCreationTime int64
	Status          int64
	Counters        string
}

type JobRunFullId struct {
	JobConfigId string
	JobRunId    string
}

func (j *JobCounters) Marshal() ([]byte, error) {
	return json.Marshal(j.counter)
}

func (j *JobCounters) Unmarshal(countersString string) error {
	j.counter = make(map[string]int64)
	err := json.Unmarshal([]byte(countersString), &(j.counter))
	return err
}

// ApplyDelta applies the changes in the given deltaObj to this.
func (j *JobCounters) ApplyDelta(deltaObj *JobCounters) {
	for key, value := range deltaObj.counter {
		j.counter[key] += value
	}
}

// GetJobStatus returns the status of the job with this JobCounterSpec.
func (j *JobCounters) GetJobStatus() int64 {
	var status int64
	if j.counter[KeyTotalTasks] == 0 {
		status = JobNotStarted
	} else if j.counter[KeyTotalTasks] == j.counter[KeyTasksCompleted] {
		status = JobSuccess
	} else if j.counter[KeyTotalTasks] == (j.counter[KeyTasksCompleted] + j.counter[KeyTasksFailed]) {
		status = JobFailed
	} else {
		status = JobInProgress
	}
	return status
}

// updateForTaskUpdate updates a JobCountersCollection's counters for the given
// TaskUpdate and oldStatus. A TaskUpdate can include an updated task, a set of newly
// created tasks, or both. These "delta" counters are eventually applied to a JobRun's
// JobCounterSpec, which keeps a running tally of the counters for the job run.
func (j *JobCountersCollection) updateForTaskUpdate(tu *TaskUpdate, oldStatus int64) error {
	// Initialize the deltas map if necessary.
	if j.deltas == nil {
		j.deltas = make(map[JobRunFullId]*JobCounters)
	}

	// Update an existing task (if applicable).
	if tu.Task != nil && tu.Task.Status != oldStatus {
		task := tu.Task
		fullJobId := task.getJobRunFullId()
		deltaObj, exists := j.deltas[fullJobId]
		if !exists {
			deltaObj = new(JobCounters)
			deltaObj.counter = make(map[string]int64)
			j.deltas[fullJobId] = deltaObj
		}
		suffix, err := CounterSuffix(task)
		if err != nil {
			return err
		}
		// Decrement the old status' counters.
		if oldStatus == Failed {
			deltaObj.counter[KeyTasksFailed] -= 1
			deltaObj.counter[KeyTasksFailed+suffix] -= 1
		} else if oldStatus == Queued {
			deltaObj.counter[KeyTasksQueued] -= 1
			deltaObj.counter[KeyTasksQueued+suffix] -= 1
		}
		// Increment the new status' counters.
		if task.Status == Success {
			deltaObj.counter[KeyTasksCompleted] += 1
			deltaObj.counter[KeyTasksCompleted+suffix] += 1
			if tu.LogEntry == nil {
				return errors.New(fmt.Sprintf(
					"Missing LogEntry for TaskUpdate: %v", tu))
			}
			le := tu.LogEntry
			switch task.TaskType {
			case listTaskType:
				deltaObj.counter[KeyListFilesFound] += le.val("files_found")
				deltaObj.counter[KeyListBytesFound] += le.val("bytes_found")
				deltaObj.counter[KeyListFileStatErrors] += le.val("file_stat_errors")
			case uploadGCSTaskType:
				deltaObj.counter[KeyBytesCopied] += le.val("src_bytes")
			}
		} else if task.Status == Failed {
			deltaObj.counter[KeyTasksFailed] += 1
			deltaObj.counter[KeyTasksFailed+suffix] += 1
		} else if task.Status == Queued {
			deltaObj.counter[KeyTasksQueued] += 1
			deltaObj.counter[KeyTasksQueued+suffix] += 1
		} else {
			return errors.New(fmt.Sprintf(
				"Found unexpected task Status in updateForTaskUpdate: %v",
				task.Status))
		}
	}

	// Add stats for newly created tasks.
	for _, task := range tu.NewTasks {
		suffix, err := CounterSuffix(task)
		if err != nil {
			return err
		}
		fullJobId := task.getJobRunFullId()
		deltaObj, exists := j.deltas[fullJobId]
		if !exists {
			deltaObj = new(JobCounters)
			deltaObj.counter = make(map[string]int64)
			j.deltas[fullJobId] = deltaObj
		}
		deltaObj.counter[KeyTotalTasks] += 1
		deltaObj.counter[KeyTotalTasks+suffix] += 1
	}

	return nil
}

// IsJobTerminated returns whether a job has terminated or not.
func IsJobTerminated(jobStatus int64) bool {
	return jobStatus == JobFailed || jobStatus == JobSuccess
}

func CounterSuffix(task *Task) (string, error) {
	switch task.TaskType {
	case listTaskType:
		return KeySuffixList, nil
	case uploadGCSTaskType:
		return KeySuffixCopy, nil
	case loadBQTaskType:
		return KeySuffixLoad, nil
	default:
		return "", errors.New(fmt.Sprintf(
			"Found unexpected TaskType updateForTaskUpdate: %v. task:%v", task.TaskType, task))
	}
}
