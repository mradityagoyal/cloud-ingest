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
	"reflect"
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
	statsDisplayTickerMaker = func() common.Ticker {
		return common.NewClockTicker(statsDisplayFreq)
	}
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
	ctrlMsgsReceived int64
	pulseMsgsSent    int64
}

type lifetimeStats struct {
	PulseStats // Embedded struct.

	taskDone    map[string]uint64
	ctrlMsgTime time.Time
	bwLimit     int64
}

// PulseStats contains stats which are sent with each Agent pulse message.
type PulseStats struct {
	CopyBytes             int64
	ListBytes             int64
	CopyOpenMs            int64
	CopyStatMs            int64
	CopySeekMs            int64
	CopyReadMs            int64
	CopyWriteMs           int64
	CopyInternalRetries   int64
	DeleteInternalRetries int64
	ListDirOpenMs         int64
	ListDirReadMs         int64
	ListFileWriteMs       int64
	ListDirWriteMs        int64
}

func (ps1 *PulseStats) add(ps2 *PulseStats) {
	ps1v := reflect.ValueOf(ps1).Elem()
	ps2v := reflect.ValueOf(ps2).Elem()
	for i := 0; i < ps1v.NumField(); i++ {
		ps1v.Field(i).SetInt(ps1v.Field(i).Int() + ps2v.Field(i).Int())
	}
}

func (ps1 *PulseStats) sub(ps2 *PulseStats) {
	ps1v := reflect.ValueOf(ps1).Elem()
	ps2v := reflect.ValueOf(ps2).Elem()
	for i := 0; i < ps1v.NumField(); i++ {
		ps1v.Field(i).SetInt(ps1v.Field(i).Int() - ps2v.Field(i).Int())
	}
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

	// For managing accumulated pulse stats.
	pulseStatsMu   sync.Mutex
	pulseStatsChan chan *PulseStats
	currPulseStats PulseStats
	prevPulseStats PulseStats

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
		pulseStatsChan:    make(chan *PulseStats, 100),
		tpTracker:         throughput.NewTracker(ctx),
		selectDone:        func() {},
		logTicker:         common.NewClockTicker(statsLogFreq),
		displayTicker:     statsDisplayTickerMaker(),
		accumulatorTicker: accumulatorTickerMaker(),
	}
	go t.track(ctx)
	return t
}

// AccumulatedPulseStats returns the PulseStats since the last time this function was called.
// This function is *NOT* idempotent, as calling it resets the underlying PulseStats.
func (t *Tracker) AccumulatedPulseStats() *PulseStats {
	if t == nil {
		return &PulseStats{}
	}
	t.pulseStatsMu.Lock()
	defer t.pulseStatsMu.Unlock()
	d := t.currPulseStats
	t.currPulseStats = PulseStats{}
	return &d
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

// CopyByteTrackingReader is an io.Reader that wraps another io.Reader and
// performs byte tracking during the Read function.
type CopyByteTrackingReader struct {
	reader  io.Reader
	tracker *Tracker
}

// NewCopyByteTrackingReader returns a CopyByteTrackingReader.
// Returns the passed in reader for a nil receiver.
func (t *Tracker) NewCopyByteTrackingReader(r io.Reader) io.Reader {
	if t == nil {
		return r
	}
	return &CopyByteTrackingReader{reader: r, tracker: t}
}

// Read implements the io.Reader interface.
func (cbtr *CopyByteTrackingReader) Read(buf []byte) (n int, err error) {
	start := time.Now()
	n, err = cbtr.reader.Read(buf)
	cbtr.tracker.pulseStatsChan <- &PulseStats{
		CopyReadMs: DurMs(start),
		CopyBytes:  int64(n),
	}
	cbtr.tracker.tpTracker.RecordBytesSent(int64(n))
	return n, err
}

// TimingReader is an io.Reader that wraps another io.Reader and
// tracks the total duration of the Read calls.
type TimingReader struct {
	reader  io.Reader
	readDur time.Duration
}

// NewTimingReader returns a TimingReader.
func NewTimingReader(r io.Reader) *TimingReader {
	return &TimingReader{reader: r}
}

// Read implements the io.Reader interface.
func (tr *TimingReader) Read(buf []byte) (n int, err error) {
	start := time.Now()
	n, err = tr.reader.Read(buf)
	tr.readDur += time.Now().Sub(start)
	return n, err
}

// ReadDur returns the total duration of this reader's Read calls.
func (tr *TimingReader) ReadDur() time.Duration {
	return tr.readDur
}

// ListByteTrackingWriter is an io.Writer that wraps another io.Writer and
// performs byte tracking during the Write function.
type ListByteTrackingWriter struct {
	writer  io.Writer
	tracker *Tracker
	file    bool
}

// NewListByteTrackingWriter returns a ListByteTrackingWriter. If 'file' is true, timing stats
// will be written for ListFileWriteMs. If false, timing stats will be written for ListDirWriteMs.
// Returns the passed in writer for a nil receiver.
func (t *Tracker) NewListByteTrackingWriter(w io.Writer, file bool) io.Writer {
	if t == nil {
		return w
	}
	return &ListByteTrackingWriter{writer: w, tracker: t, file: file}
}

// Write implements the io.Writer interface.
func (lbtw *ListByteTrackingWriter) Write(p []byte) (n int, err error) {
	start := time.Now()
	n, err = lbtw.writer.Write(p)
	ps := &PulseStats{ListBytes: int64(n)}
	if lbtw.file {
		ps.ListFileWriteMs = DurMs(start)
	} else {
		ps.ListDirWriteMs = DurMs(start)
	}
	lbtw.tracker.pulseStatsChan <- ps
	return n, err
}

// RecordPulseStats tracks stats contained within 'ps'.
// Takes no action for a nil receiver.
func (t *Tracker) RecordPulseStats(ps *PulseStats) {
	if t == nil {
		return
	}
	t.pulseStatsChan <- ps
}

// DurMs returns the duration in millis between time.Now() and 'start'.
func DurMs(start time.Time) int64 {
	return time.Now().Sub(start).Nanoseconds() / 1000000
}

// RecordBWLimit tracks the current bandwidth limit.
// Takes no action for a nil receiver.
func (t *Tracker) RecordBWLimit(agentBW int64) {
	if t == nil {
		return
	}
	t.bwLimitChan <- agentBW
}

// RecordCtrlMsg tracks received control messages.
// Takes no action for a nil receiver.
func (t *Tracker) RecordCtrlMsg(time time.Time) {
	if t == nil {
		return
	}
	t.ctrlMsgChan <- time
}

// RecordPulseMsg tracks sent pulse messages.
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
		case ps := <-t.pulseStatsChan:
			t.periodic.bytesCopied += ps.CopyBytes
			t.lifetime.PulseStats.add(ps)
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
			t.accumulatePulseStats()
		}
		t.selectDone() // Testing hook.
	}
}

func (t *Tracker) accumulatePulseStats() {
	t.pulseStatsMu.Lock()
	defer t.pulseStatsMu.Unlock()
	t.currPulseStats.add(&t.lifetime.PulseStats)
	t.currPulseStats.sub(&t.prevPulseStats)
	t.prevPulseStats = t.lifetime.PulseStats
}

func (t *Tracker) displayStats() string {
	// Generate the transmission rate and sum.
	txRate := fmt.Sprintf("txRate:%v/s", byteCountBinary(t.tpTracker.Throughput(), 7))
	if txLim := t.lifetime.bwLimit; txLim > 0 && txLim < math.MaxInt32 {
		txRate += fmt.Sprintf(" (capped at %v/s)", byteCountBinary(t.lifetime.bwLimit, 7))
	}
	txSum := fmt.Sprintf("txSum:%v", byteCountBinary(t.lifetime.CopyBytes, 7))

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
