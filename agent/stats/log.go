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
	"fmt"
	"regexp"
	"time"

	"github.com/golang/glog"
)

var (
	// The named groups in this regex (e.g. ?P<copyDone>) yield the column names through SubexpNames().
	// Example log line that this will match:
	//   copy done:200 fail:1 dur:1.234,4.567,2.345 list done:1 fail:0 dur:0.123,0.123,0.123 txBytes:123456 ctrlMsgs:12 pulseMsgs:12
	msgRE = regexp.MustCompile(`` +
		`copy ` +
		`done:(?P<copyDone>\d+) ` +
		`fail:(?P<copyFail>\d+) ` +
		`dur:(?P<copyDurMin>\d+\.?\d*),(?P<copyDurMax>\d+\.?\d*),(?P<copyDurAvg>\d+\.?\d*) ` +
		`list ` +
		`done:(?P<listDone>\d+) ` +
		`fail:(?P<listFail>\d+) ` +
		`dur:(?P<listDurMin>\d+\.?\d*),(?P<listDurMax>\d+\.?\d*),(?P<listDurAvg>\d+\.?\d*) ` +
		`txBytes:(?P<txBytes>\d+) ` +
		`ctrlMsgs:(?P<ctrlMsgs>\d+) ` +
		`pulseMsgs:(?P<pulseMsgs>\d+)`,
	)
)

func (p *periodicStats) reset() {
	p.taskDurations = make(map[string][]time.Duration)
	p.taskFailures = make(map[string]int)
	p.bytesCopied = 0
	p.ctrlMsgsReceived = 0
	p.pulseMsgsSent = 0
}

func (p *periodicStats) glogAndReset() string {
	if len(p.taskDurations) == 0 && len(p.taskFailures) == 0 && p.bytesCopied == 0 {
		// Exit early if there's no info besides control/pulse msgs. This will
		// accumulate ctrlMsgsReceived and pulseMsgsSent, but that's preferrable
		// to spamming the logs with many mostly empty lines.
		return ""
	}
	var msg string
	for i, k := range []string{"copy", "list"} {
		var min, max, avg time.Duration
		durs := p.taskDurations[k]
		if durs != nil || len(durs) != 0 {
			min = durs[0]
			max = durs[0]
			var sum time.Duration
			for _, d := range durs {
				sum += d
				if d > max {
					max = d
				}
				if d < min {
					min = d
				}
			}
			avg = sum / time.Duration(len(durs))
		}
		// Convert the durations to a float64 of seconds, otherwise printing
		// a time.Duration will automatically convert the units, which we don't want.
		// Graphing "1.234s" alongside "650ms" is less convenient than just
		// "1.234" and "0.650".
		minS := float64(min.Truncate(time.Millisecond)) / float64(time.Second)
		maxS := float64(max.Truncate(time.Millisecond)) / float64(time.Second)
		avgS := float64(avg.Truncate(time.Millisecond)) / float64(time.Second)

		if i > 0 {
			msg += " "
		}
		msg += fmt.Sprintf("%s done:%v fail:%v dur:%.3f,%.3f,%.3f", k, len(durs), p.taskFailures[k], minS, maxS, avgS)
	}
	msg += fmt.Sprintf(" txBytes:%v", p.bytesCopied)
	msg += fmt.Sprintf(" ctrlMsgs:%v", p.ctrlMsgsReceived)
	msg += fmt.Sprintf(" pulseMsgs:%v", p.pulseMsgsSent)
	p.reset()

	glog.Info(msg)
	return msg // Returned for testing only.
}

// ParseLogMsg parses a log msg string and returns the corresponding columns and values. The log msg
// must be of the format as defined by glogAndReset and msgRE. If the log msg is not of this format, the
// returned col and vals slices will be nil.
func ParseLogMsg(msg string) (cols, vals []string) {
	match := msgRE.FindStringSubmatch(msg)
	if match == nil {
		return nil, nil
	}
	for i, m := range match[1:] {
		cols = append(cols, msgRE.SubexpNames()[i+1])
		vals = append(vals, m)
	}
	return cols, vals
}
