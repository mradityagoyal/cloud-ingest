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

package rate

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"

	"golang.org/x/time/rate"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
)

var (
	mu       sync.RWMutex     // Protects jobRunBW and projectBWLimiter.
	jobRunBW map[string]int64 // JobrunRelRsrcName to bandwidth mapping.

	// Project-wide bandwidth limiter.
	projectBWLimiter = rate.NewLimiter(rate.Limit(math.MaxInt64), math.MaxInt32)
)

// ProcessCtrlMsg updates the jobRunBW mapping and projectBWLimiter given the values
// in the control message.
func ProcessCtrlMsg(cm *controlpb.Control, st *stats.Tracker) {
	// Currently, we do not have a way to set per job run BW control. The API only
	// allows setting project level BW. For future extensions, DCP distributes the
	// total project BW over the active job runs. Here we aggregate it again to control
	// the BW on a project level.
	jrBW := make(map[string]int64)
	var projectBW int64
	for _, jobBW := range cm.JobRunsBandwidths {
		jrBW[jobBW.JobrunRelRsrcName] = jobBW.Bandwidth
		projectBW += jobBW.Bandwidth
	}
	mu.Lock()
	defer mu.Unlock()
	jobRunBW = jrBW
	if diff := math.Abs(float64(projectBW) - float64(projectBWLimiter.Limit())); diff > 0.0000001 {
		burst := math.MaxInt32
		if projectBW < int64(burst) {
			burst = int(projectBW)
		}
		projectBWLimiter = rate.NewLimiter(rate.Limit(projectBW), burst)
		if st != nil {
			st.RecordBWLimit(projectBW)
		}
	}
}

// IsJobRunActive returns a bool indicating if a job run is active (paused).
func IsJobRunActive(jobrunRelRsrcName string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return jobRunBW[jobrunRelRsrcName] != 0
}

// RateLimitingReader is an io.Reader that wraps another io.Reader and
// enforces rate limiting during the Read function.
type RateLimitingReader struct {
	reader io.Reader
}

// NewRateLimitingReader returns a RateLimitingReader.
func NewRateLimitingReader(r io.Reader) io.Reader {
	return RateLimitingReader{reader: r}
}

// Read implements the io.Reader interface.
func (rlr RateLimitingReader) Read(buf []byte) (n int, err error) {
	// Shrink the read buf if necessary. This ensures the read doesn't just
	// block for one massive copy, and instead hands out data every second.
	mu.RLock()
	lim := int(projectBWLimiter.Limit())
	mu.RUnlock()
	if 0 < lim && lim < len(buf) {
		buf = buf[0:lim]
	}

	// Perform the read.
	if n, err = rlr.reader.Read(buf); err != nil {
		return 0, err
	}

	// Enforce the rate limit.
	mu.RLock()
	r := projectBWLimiter.ReserveN(time.Now(), n)
	mu.RUnlock()
	if r.OK() {
		time.Sleep(r.Delay())
	}

	return n, nil
}
