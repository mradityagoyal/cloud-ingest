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
	"fmt"
	"io"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/common"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats/throughput"
	"github.com/golang/glog"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const (
	statsLogFreq     = 3 * time.Minute // The frequency of logging periodic stats.
	statsDisplayFreq = 1 * time.Second // The frequency of displaying stats to stdout.
	accumulatorFreq  = 1 * time.Second // The frequency of accumulating bytes copied.
)

var (
	accumulatorTickerMaker = func() common.Ticker {
		return common.NewClockTicker(accumulatorFreq)
	}
)

type taskDur struct {
	task string
	dur  time.Duration
}

type periodicStats struct {
	taskDurations    map[string][]time.Duration
	taskFailures     map[string]int
	bytesCopied      int64
	listBytesCopied  int64
	ctrlMsgsReceived int64
	pulseMsgsSent    int64
}

type lifetimeStats struct {
	taskDone        map[string]uint64
	bytesCopied     int64
	listBytesCopied int64

	ctrlMsgTime time.Time
	bwLimit     int64
}

type accumulator struct {
	mu               sync.Mutex
	bytesSentChan    chan int64
	accumulatedBytes int64
	prevBytes        int64
}

// Tracker collects stats about the Agent, provides a display to STDOUT, and
// periodically writes to the INFO log. Stats are collected by calling the
// various Record* functions as appropriate.
type Tracker struct {
	taskDurChan  chan taskDur   // Channel to record task durations.
	taskFailChan chan string    // Channel to record task failures.
	bwLimitChan  chan int64     // Channel to record the bandwidth limit.
	ctrlMsgChan  chan time.Time // Channel to record control message timing.
	pulseMsgChan chan struct{}  // Channel to record send pulse messages.

	periodic  periodicStats       // Reset after every time they're INFO logged.
	lifetime  lifetimeStats       // Cumulative for the lifetime of this procces.
	tpTracker *throughput.Tracker // Measures outgoing copy throughput.

	spinnerIdx int // For displaying the mighty spinner.

	// These fields support func AccumulatedBytes, which allows the
	// PulseSender to send the count of transferred bytes with each pulse.
	copyBytes accumulator
	listBytes accumulator

	// Testing hooks.
	selectDone        func()
	logTicker         common.Ticker
	displayTicker     common.Ticker
	accumulatorTicker common.Ticker
}

// NewTracker returns a new Tracker, which can then be used to record stats.
func NewTracker(ctx context.Context) *Tracker {
	t := &Tracker{
		// Large buffers to avoid blocking.
		taskDurChan:  make(chan taskDur, 100),
		taskFailChan: make(chan string, 10),
		bwLimitChan:  make(chan int64, 10),
		ctrlMsgChan:  make(chan time.Time, 10),
		pulseMsgChan: make(chan struct{}, 10),
		periodic: periodicStats{
			taskDurations: make(map[string][]time.Duration),
			taskFailures:  make(map[string]int),
		},
		lifetime: lifetimeStats{
			taskDone:    map[string]uint64{"copy": 0, "list": 0},
			ctrlMsgTime: time.Now(),
			bwLimit:     math.MaxInt32,
		},
		copyBytes:         accumulator{bytesSentChan: make(chan int64, 100)},
		listBytes:         accumulator{bytesSentChan: make(chan int64, 100)},
		tpTracker:         throughput.NewTracker(ctx),
		selectDone:        func() {},
		logTicker:         common.NewClockTicker(statsLogFreq),
		displayTicker:     common.NewClockTicker(statsDisplayFreq),
		accumulatorTicker: accumulatorTickerMaker(),
	}
	go t.track(ctx)
	return t
}

// AccumulatedBytes returns the number of bytesCopied since the last call to
// this function. This function is *NOT* idempotent, as calling it resets the
// accumulatedBytes.
//
// Returns zero for a nil receiver.
func (a *accumulator) AccumulatedBytes() int64 {
	if a == nil {
		return 0
	}
	a.mu.Lock()
	// defers stack, set to 0 will happen before the mutex unlocks.
	defer a.mu.Unlock()
	defer func() { a.accumulatedBytes = 0 }()
	return a.accumulatedBytes
}

// AccumulatedCopyBytes convenience function for CopyBytes.AccumulatedBytes()
func (t *Tracker) AccumulatedCopyBytes() int64 {
	if t == nil {
		return 0
	}
	return t.copyBytes.AccumulatedBytes()
}

// AccumulatedListBytes convenience function for ListBytes.AccumulatedBytes()
func (t *Tracker) AccumulatedListBytes() int64 {
	if t == nil {
		return 0
	}
	return t.listBytes.AccumulatedBytes()
}

// RecordTaskResp tracks the count and duration of completed tasks.
//
// Takes no action for a nil receiver.
func (t *Tracker) RecordTaskResp(resp *taskpb.TaskRespMsg, dur time.Duration) {
	if t == nil {
		return
	}
	task := ""
	if resp.ReqSpec.GetCopySpec() != nil {
		task = "copy"
	} else if resp.ReqSpec.GetListSpec() != nil {
		task = "list"
	} else if resp.ReqSpec.GetCopyBundleSpec() != nil {
		task = "copy"
	} else {
		glog.Errorf("resp.ReqSpec doesn't match any known spec type: %v", resp.ReqSpec)
	}

	if task != "" {
		t.taskDurChan <- taskDur{task, dur} // Record the task duration.

		if resp.FailureType != taskpb.FailureType_UNSET_FAILURE_TYPE {
			t.taskFailChan <- task // Record the failure.
		}
	}
}

// ByteTrackingReader is an io.Reader that wraps another io.Reader and
// performs byte tracking during the Read function.
type ByteTrackingReader struct {
	reader  io.Reader
	tracker *Tracker
}

// NewByteTrackingReader returns a ByteTrackingReader.
//
// Returns the passed in reader for a nil receiver.
func (t *Tracker) NewByteTrackingReader(r io.Reader) io.Reader {
	if t == nil {
		return r
	}
	return ByteTrackingReader{reader: r, tracker: t}
}

// Read implements the io.Reader interface.
func (btr ByteTrackingReader) Read(buf []byte) (n int, err error) {
	if n, err = btr.reader.Read(buf); err != nil {
		return 0, err
	}
	btr.tracker.RecordCopyBytesSent(int64(n))
	return n, nil
}

// ByteTrackingWriter is an io.Writer that records how many bytes are written
// and adds them up to the throughput tracker.
type ByteTrackingWriter struct {
	writer  io.Writer
	tracker *Tracker
}

// NewByteTrackingWriter returns a ByteTrackingWriter.
func (t *Tracker) NewByteTrackingWriter(w io.Writer) io.Writer {
	if t == nil {
		return nil
	}
	return ByteTrackingWriter{writer: w, tracker: t}
}

// Write implements the io.Writer interface.
func (btw ByteTrackingWriter) Write(p []byte) (n int, err error) {
	nb := len(p)
	btw.tracker.RecordCopyBytesSent(int64(nb))
	btw.writer.Write(p)
	return nb, nil
}

// RecordCopyBytesSent tracks the count of bytes sent, and enables throughput tracking.
// For accurate throughput measurement this function should be called every time
// bytes are sent on the wire. More frequent and granular calls to this function
// will provide a more accurate throughput measurement.
//
// Takes no action for a nil receiver.
func (t *Tracker) RecordCopyBytesSent(bytes int64) {
	if t == nil {
		return
	}
	t.tpTracker.RecordBytesSent(bytes)
	t.copyBytes.bytesSentChan <- bytes
}

// RecordListBytesSent tracks the count of list bytes sent
// (analogous to RecordCopyBytesSent).
//
// Takes no action for a nil receiver.
func (t *Tracker) RecordListBytesSent(bytes int64) {
	if t == nil {
		return
	}
	t.listBytes.bytesSentChan <- bytes
}

// RecordBWLimit tracks the current bandwidth limit.
//
// Takes no action for a nil receiver.
func (t *Tracker) RecordBWLimit(agentBW int64) {
	if t == nil {
		return
	}
	t.bwLimitChan <- agentBW
}

// RecordCtrlMsg tracks received control messages.
//
// Will take no action if the receiver is nil.
func (t *Tracker) RecordCtrlMsg(time time.Time) {
	if t == nil {
		return
	}
	t.ctrlMsgChan <- time
}

// RecordPulseMsg tracks sent pulse messages.
//
// Takes no action for a nil receiver.
func (t *Tracker) RecordPulseMsg() {
	if t == nil {
		return
	}
	t.pulseMsgChan <- struct{}{}
}

func (t *Tracker) track(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				glog.Infof("stats.Tracker track ctx ended with err: %v", err)
			}
			return
		case tr := <-t.taskDurChan:
			t.periodic.taskDurations[tr.task] = append(t.periodic.taskDurations[tr.task], tr.dur)
			t.lifetime.taskDone[tr.task]++
		case task := <-t.taskFailChan:
			t.periodic.taskFailures[task]++
		case copyBytes := <-t.copyBytes.bytesSentChan:
			t.periodic.bytesCopied += copyBytes
			t.lifetime.bytesCopied += copyBytes
		case listBytes := <-t.listBytes.bytesSentChan:
			t.periodic.listBytesCopied += listBytes
			t.lifetime.listBytesCopied += listBytes
		case agentBW := <-t.bwLimitChan:
			t.lifetime.bwLimit = agentBW
		case time := <-t.ctrlMsgChan:
			t.periodic.ctrlMsgsReceived++
			t.lifetime.ctrlMsgTime = time
		case <-t.pulseMsgChan:
			t.periodic.pulseMsgsSent++
		case <-t.logTicker.GetChannel():
			t.periodic.glogAndReset()
		case <-t.displayTicker.GetChannel():
			t.displayStats()
		case <-t.accumulatorTicker.GetChannel():
			t.accumulateByteStats()
		}
		t.selectDone() // Testing hook.
	}
}

func (t *Tracker) accumulateByteStats() {
	t.copyBytes.mu.Lock()
	t.copyBytes.accumulatedBytes += t.lifetime.bytesCopied - t.copyBytes.prevBytes
	t.copyBytes.prevBytes = t.lifetime.bytesCopied
	t.copyBytes.mu.Unlock()
	t.listBytes.mu.Lock()
	t.listBytes.accumulatedBytes += t.lifetime.listBytesCopied - t.listBytes.prevBytes
	t.listBytes.prevBytes = t.lifetime.listBytesCopied
	t.listBytes.mu.Unlock()
}

func (t *Tracker) displayStats() string {
	// Generate the transmission rate and sum.
	txRate := fmt.Sprintf("txRate:%v/s", byteCountBinary(t.tpTracker.Throughput(), 7))
	if txLim := t.lifetime.bwLimit; txLim > 0 && txLim < math.MaxInt32 {
		txRate += fmt.Sprintf(" (capped at %v/s)", byteCountBinary(t.lifetime.bwLimit, 7))
	}
	txSum := fmt.Sprintf("txSum:%v", byteCountBinary(t.lifetime.bytesCopied, 7))

	// Generate the task response counts.
	taskResps := "taskResps["
	var keys []string
	for k := range t.lifetime.taskDone {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			taskResps += " "
		}
		taskResps += fmt.Sprintf("%v:%v", k, t.lifetime.taskDone[k])
	}
	taskResps += "]"

	// Generate the control message age and status.
	ctrlMsgAge := "-"
	ctrlMsgHealth := "-"
	if !t.lifetime.ctrlMsgTime.IsZero() {
		age := time.Now().Sub(t.lifetime.ctrlMsgTime).Truncate(time.Second)
		ctrlMsgAge = fmt.Sprintf("%v", age)
		ctrlMsgHealth = "ok"
		if age > 30*time.Second {
			ctrlMsgHealth = "??"
		}
	}
	ctrlMsg := fmt.Sprintf("ctrlMsgAge:%v (%v)", ctrlMsgAge, ctrlMsgHealth)

	// Generate the spinner.
	spinnerChars := `-\|/`
	t.spinnerIdx = (t.spinnerIdx + 1) % len(spinnerChars)
	spinner := spinnerChars[t.spinnerIdx]

	// Display the generated stats.
	// TODO(b/123023481): Ensure the Agent display works on Windows.
	fmt.Printf("\r%120s\r", "") // Overwrite the previous line and reset the cursor.
	displayLine := fmt.Sprintf("%v %v %v %v %c", txRate, txSum, taskResps, ctrlMsg, spinner)
	fmt.Print(displayLine)
	return displayLine // For testing.
}

func byteCountBinary(b int64, pad int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%*dB", pad, b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%*.1f%ciB", pad-2, float64(b)/float64(div), "KMGTPE"[exp])
}
