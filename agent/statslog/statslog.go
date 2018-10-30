/*
Copyright 2018 Google Inc. All Rights Reserved.
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

package statslog

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/golang/glog"
)

var (
	statsLogFreq = flag.Duration("stats-log-freq", 3*time.Minute, "The frequency of logging handler timing stats")
)

type StatsLog struct {
	mu     sync.Mutex
	stats  map[string][]time.Duration
	ticker *time.Ticker
}

func New() *StatsLog {
	sl := &StatsLog{
		stats:  make(map[string][]time.Duration),
		ticker: time.NewTicker(*statsLogFreq),
	}
	return sl
}

func (sl *StatsLog) PeriodicallyLogStats(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				glog.Infof("PeriodicallyLogStats ctx ended with err: %v", err)
			}
			return
		case <-sl.ticker.C:
			sl.calcStatsAndLog()
		}
	}
}

func (sl *StatsLog) AddSample(msgType string, t time.Duration) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.stats[msgType] = append(sl.stats[msgType], t)
}

func (sl *StatsLog) calcStatsAndLog() string {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if len(sl.stats) == 0 {
		return ""
	}
	logLine := "type(count)[time min,max,avg]:"
	var keys []string
	for k := range sl.stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		times := sl.stats[k]
		max := times[0]
		min := times[0]
		var avg time.Duration
		for _, t := range times {
			avg += t
			if t > max {
				max = t
			}
			if t < min {
				min = t
			}
		}
		avg /= time.Duration(len(times))
		min = min.Truncate(1 * time.Millisecond)
		max = max.Truncate(1 * time.Millisecond)
		avg = avg.Truncate(1 * time.Millisecond)
		logLine += fmt.Sprintf("\n\t%s(%d)[%v,%v,%v]", k, len(times), min, max, avg)
	}
	glog.Info(logLine)
	sl.stats = make(map[string][]time.Duration)
	return logLine // For testing.
}
