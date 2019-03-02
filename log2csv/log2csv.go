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

// This helper tool is used to parse glog files produced by the Agent. It will extract the
// periodically produced "stats" log lines, giving insight into the Agent's behavior and
// performance. The output from this tool is a csv file which can easily be imported into
// your favorite speadsheet program for easy analysis.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
)

var (
	log = flag.String("log", "/tmp/agentmain.INFO", "The source log file to parse.")
	csv = flag.String("csv", "/tmp/agentmain.INFO.csv", "The target csv file to write to.")
)

func main() {
	flag.Parse()

	logFile, err := os.Open(*log)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	fmt.Printf("opened %v for read\n", *log)

	csvFile, err := os.Create(*csv)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	fmt.Printf("opened %v for write\n", *csv)

	colsWritten := false
	linesWritten := 0
	reader := bufio.NewReader(logFile)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("ReadString got err: ", err)
			}
			break
		}
		cols, vals := parseLogLine(line)
		if len(cols) == 0 || len(vals) == 0 {
			continue
		}

		if !colsWritten {
			csvFile.WriteString(strings.Join(cols, ",") + "\n")
			colsWritten = true
		}
		csvFile.WriteString(strings.Join(vals, ",") + "\n")
		linesWritten++
		fmt.Printf(".")
	}
	fmt.Printf("\nwrote %d lines to %s\n", linesWritten, *csv)
}

func parseLogLine(logLine string) (cols, vals []string) {
	// The logLine consists of two parts, the header (added by glog), and the msg (what we wrote).
	// The header format is defined in github.com/golang/glog/glog.go.

	// Extract the log header.
	cols = append(cols, "month")
	vals = append(vals, logLine[1:3])
	cols = append(cols, "day")
	vals = append(vals, logLine[3:5])
	cols = append(cols, "time")
	vals = append(vals, logLine[6:21])

	// Extract the log msg.
	msg := logLine[strings.Index(logLine, "]")+1:]
	msgCols, msgVals := stats.ParseLogMsg(msg)
	if len(msgCols) == 0 || len(msgVals) == 0 {
		return nil, nil
	}
	cols = append(cols, msgCols...)
	vals = append(vals, msgVals...)

	return cols, vals
}
