/* Copyright 2017 Google Inc. All Rights Reserved.
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
	"fmt"
	"hash/fnv"

	"cloud.google.com/go/spanner"
)

type LogEntry struct {
	data map[string]interface{}
}

func NewLogEntry(data map[string]interface{}) *LogEntry {
	logEntry := new(LogEntry)
	logEntry.data = data
	return logEntry
}

func (le LogEntry) val(key string) int64 {
	value, err := le.data[key].(json.Number).Int64()
	if err != nil {
		return int64(0)
	}
	return value
}

func (le LogEntry) String() string {
	return fmt.Sprint(le.data)
}

// Returns an array of LogEntries table columns.
func getLogEntryInsertColumns() []string {
	return []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"LogEntryId",
		"LogEntryCreationTime",
		"CurrentStatus",
		"PreviousStatus",
		"FailureMessage",
		"LogEntry",
	}
}

// Adds a mutation to 'mutations' which inserts a LogEntry for the given task.
func insertLogEntryMutation(mutations *[]*spanner.Mutation, task *Task, previousStatus int64, logEntry *LogEntry, timestamp int64) {
	logEntryString := fmt.Sprint(logEntry)
	h := fnv.New64a()
	h.Write([]byte(logEntryString))
	h.Write([]byte(fmt.Sprintln(timestamp)))
	logEntryId := int64(h.Sum64())

	*mutations = append(*mutations, spanner.Insert("LogEntries", getLogEntryInsertColumns(),
		[]interface{}{
			task.JobConfigId,
			task.JobRunId,
			task.TaskId,
			logEntryId,
			timestamp,
			task.Status,
			previousStatus,
			task.FailureMessage,
			logEntryString,
		}))
}
