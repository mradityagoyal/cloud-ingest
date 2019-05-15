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

package stats

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func TestLogMsgFormatAndParse(t *testing.T) {
	type sample struct {
		trm *taskpb.TaskRespMsg
		d   time.Duration
	}
	samples := []sample{
		sample{copyTaskRespMsg, 1 * time.Millisecond},
		sample{copyTaskRespMsg, 2 * time.Millisecond},
		sample{copyTaskRespMsg, 3 * time.Millisecond},
		sample{listTaskRespMsg, 4 * time.Millisecond},
		sample{listTaskRespMsg, 5 * time.Millisecond},
		sample{listTaskRespMsg, 6 * time.Millisecond},
	}
	st := NewTracker(context.Background())
	var wg sync.WaitGroup
	st.selectDone = func() { wg.Done() } // The test hook.
	for i, s := range samples {
		wg.Add(4)
		st.RecordTaskResp(s.trm, s.d)
		st.RecordCopyBytesSent(int64(i))
		st.RecordCtrlMsg(time.Now())
		st.RecordPulseMsg()
		wg.Wait() // Force the Tracker to collect the recorded stats.
	}
	logMsg := st.periodic.glogAndReset()
	if x := len(st.periodic.taskDurations); x != 0 {
		t.Errorf("after reset len(st.periodic.taskDurations) = %v, want 0", x)
	}
	if x := len(st.periodic.taskFailures); x != 0 {
		t.Errorf("after reset len(st.periodic.taskFailures) = %v, want 0", x)
	}
	if x := st.periodic.bytesCopied; x != 0 {
		t.Errorf("after reset st.periodic.bytesCopied = %v, want 0", x)
	}
	if x := st.periodic.ctrlMsgsReceived; x != 0 {
		t.Errorf("after reset st.periodic.ctrlMsgsReceived = %v, want 0", x)
	}
	if x := st.periodic.pulseMsgsSent; x != 0 {
		t.Errorf("after reset st.periodic.pulseMsgsSent= %v, want 0", x)
	}

	gotCols, gotVals := ParseLogMsg(logMsg)
	wantCols := []string{"copyDone", "copyFail", "copyDurMin", "copyDurMax", "copyDurAvg", "listDone", "listFail", "listDurMin", "listDurMax", "listDurAvg", "txBytes", "ctrlMsgs", "pulseMsgs"}
	wantVals := []string{"3", "0", "0.001", "0.003", "0.002", "3", "0", "0.004", "0.006", "0.005", "15", "6", "6"}
	if !cmp.Equal(gotCols, wantCols) {
		t.Errorf("pargeLogMsg(%q) got cols %v, want %v", logMsg, gotCols, wantCols)
	}
	if !cmp.Equal(gotVals, wantVals) {
		t.Errorf("pargeLogMsg(%q) got vals %v, want %v", logMsg, gotVals, wantVals)
	}
}
