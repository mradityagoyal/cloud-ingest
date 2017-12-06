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
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
)

// This channel is used to notify the LogEntryProcessor that a job run changed
// status, so the LogEntryProcessor should attempt to process logs.
var JobRunStatusChangeNotificationChannel chan int

const (
	logEntryCountCheckInterval time.Duration = 1 * time.Minute
	noProgressTimeout          time.Duration = 60 * time.Minute
	maxNoProgressCount         int64         = int64(noProgressTimeout / logEntryCountCheckInterval)
	numLogsToFetchPerRun       int64         = 10000

	// Similar to the format specified in RFC 3339 (used by
	// time.MarshalText), however this format uses trailing zeros for the
	// nanoseconds, so we get a consistent length for time in the log line.
	// This makes the logs easier to visually parse.  For referenece, here
	// is the RFC 3339 format.
	// RFC3339fmt = "2006-01-02T15:04:05.999999999Z07:00"
	logTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"
)

// *****************************************************************************
// LogEntry
// *****************************************************************************
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
		"CreationTime",
		"CurrentStatus",
		"PreviousStatus",
		"FailureMessage",
		"LogEntry",
		"Processed",
	}
}

// Adds a mutation to 'mutations' which inserts a LogEntry for the given task.
func insertLogEntryMutation(mutations *[]*spanner.Mutation, task *Task, previousStatus int64, logEntry *LogEntry, timestamp int64) {
	var logEntryString string
	if logEntry != nil && logEntry.data != nil {
		// Sort the logEntry's map's keys to ensure a stable order (something Go
		// doesn't guarantee).
		var keys []string
		for k, _ := range logEntry.data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var kv []string
		for _, k := range keys {
			kv = append(kv, fmt.Sprintf("%v:%v", k, logEntry.data[k]))
		}
		logEntryString = strings.Join(kv, " ")
	}

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
			false, // All entries start out un-processed.
		}))
}

// *****************************************************************************
// LogEntryProcessor
// *****************************************************************************

type LogEntryRow struct {
	JobConfigId    string
	JobRunId       string
	TaskId         string
	LogEntryId     int64
	CreationTime   int64
	CurrentStatus  int64
	PreviousStatus int64
	FailureMessage string
	LogEntry       string
	Processed      bool
}

// sanitizeFailureMessage replaces characters that would make post processing
// log entries from a file difficult. It is intended as an accompanying func
// to the LogEntryRow's Stringer implementation of String().
func sanitizeFailureMessage(s string) string {
	// \n is replaced since newlines would make a single log entry span
	// multiple lines.
	s = strings.Replace(s, "\n", "\\n", -1)
	// ' is replaced, since it is used to delimit the FailureMessage and
	// WorkerLog in the LogEntry.
	s = strings.Replace(s, "'", "`", -1)
	return s
}

func (l LogEntryRow) String() string {
	timebytes := time.Unix(0, l.CreationTime).Format(logTimeFormat)
	return fmt.Sprintf("%v %v %v->%v FailureMessage:'%v' WorkerLog:'%v'",
		string(timebytes), l.TaskId,
		proto.TaskStatus_Type_name[int32(l.PreviousStatus)],
		proto.TaskStatus_Type_name[int32(l.CurrentStatus)],
		sanitizeFailureMessage(l.FailureMessage), l.LogEntry)
}

type LogEntryProcessor struct {
	Gcs   gcloud.GCS // For writing the log entry files to GCS.
	Store Store      // For reading and updating LogEntries in Spanner.
}

// TODO(b/69171420): This assumes a single processor and single DCP all in the same
// binary. We will need to revisit this triggering scheme when we figure out how to
// support multiple DCPs for a single project.
func (lep LogEntryProcessor) ProcessLogs() {
	t := NewClockTicker(logEntryCountCheckInterval)
	if JobRunStatusChangeNotificationChannel == nil {
		// Buffer the channel so the sender doesn't block.
		JobRunStatusChangeNotificationChannel = make(chan int, 10)
	}
	go lep.continuouslyProcessLogs(context.Background(), t, JobRunStatusChangeNotificationChannel, nil)
}

func (lep LogEntryProcessor) continuouslyProcessLogs(
	ctx context.Context, t Ticker, jobrunChannel chan int, testChannel chan int) {
	periodicCheck := t.GetChannel()
	var lastN, noProgressCount int64
	for {
		select {
		case <-periodicCheck:
			// Dumps logs when enough have accumulated.
			n, err := lep.Store.GetNumUnprocessedLogs()
			if err != nil {
				log.Println("Error getting numUnprocessedLogs:", err)
				continue
			}
			if n >= numLogsToFetchPerRun {
				lep.SingleLogsProcessingRun(ctx, numLogsToFetchPerRun)
			} else if n > 0 {
				// Also, dump logs if there has been no progress (the
				// number of log entries hasn't changed) for too long.
				if n == lastN {
					noProgressCount++
				} else {
					lastN = n
					noProgressCount = 0
				}
				if noProgressCount >= maxNoProgressCount {
					lep.SingleLogsProcessingRun(ctx, numLogsToFetchPerRun)
					noProgressCount = 0
				}
			}
		case <-jobrunChannel:
			// A job run status change (e.g. to success) will process logs so
			// the user doesn't have to wait for those logs to show up in GCS.
			// This should take care of the tail of logs at the end of a job.
			n, err := lep.Store.GetNumUnprocessedLogs()
			if err != nil {
				log.Println("Error getting numUnprocessedLogs:", err)
				continue
			}
			if n > 0 {
				lep.SingleLogsProcessingRun(ctx, numLogsToFetchPerRun)
			}
		}
		if testChannel != nil {
			testChannel <- 0
		}
	}
}

// SingleLogsProcessingRun pulls up to 'n' unprocessed rows from the LogEntries table,
// writes them to GCS, and then marks those rows as processed. Note that writing to GCS
// and updating the LogEntries rows in Spanner is not transactional, so it's possible
// that the same log entry will be written to GCS more than once.
func (lep LogEntryProcessor) SingleLogsProcessingRun(ctx context.Context, n int64) {
	logs, err := lep.Store.GetUnprocessedLogs(n)
	if err != nil {
		log.Println("Error fetching unprocessed logs:", err)
		return
	}
	if len(logs) == 0 {
		log.Println("Found no logs to process.")
		return
	}

	// Map of JobConfigId to the corresonding GCS log file.
	gcsLogFiles := make(map[string]io.WriteCloser)

	// TODO(b/69123303): Investigate different file grouping schemes.
	// Write the logs to their respective files.
	for _, logEntryRow := range logs {
		jobConfigId := logEntryRow.JobConfigId
		gcsLogFile, ok := gcsLogFiles[jobConfigId]
		if !ok {
			jobSpec, err := lep.Store.GetJobSpec(jobConfigId)
			if err != nil {
				// TODO(b/69171696): We need to figure out how to
				// grabefully handle this situation.
				log.Println("Error getting JobSpec:", err)
				return
			}
			bucketName := jobSpec.GCSBucket
			timebytes := time.Unix(0, logEntryRow.CreationTime).Format(logTimeFormat)
			// Note that the timestamp is in nanoseconds, so collisions are
			// nearly impossible (two subsequent Go timestamps are ~50ns apart).
			objectName := fmt.Sprintf("logs/%v/%v.log", jobConfigId, string(timebytes))
			gcsLogFile = lep.Gcs.NewWriter(ctx, bucketName, objectName)
			gcsLogFiles[jobConfigId] = gcsLogFile
		}
		_, err := fmt.Fprintln(gcsLogFile, logEntryRow)
		if err != nil {
			log.Println("Error writing to the GCS log file:", err)
			return
		}
	}

	// Close all the open files so the logs get written to GCS.
	for _, gcsLogFile := range gcsLogFiles {
		if err := gcsLogFile.Close(); err != nil {
			log.Println("Error closing GCS object:", err)
			return
		}
	}

	// Mark the logs as processed.
	err = lep.Store.MarkLogsAsProcessed(logs)
	if err != nil {
		log.Println("Error marking logs as processed:", err)
	}
}
