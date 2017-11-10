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

// Binary to run the cloud ingest load testing.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/tests/perf"
)

func main() {
	protoMessagePath := flag.String("msg-file", "", "Path of the proto message file.")
	projectId := flag.String(
		"project-id", "",
		"Project id to run the perf. Empty project will choose the default project.")
	updateInterval := flag.Duration("update-interval", 5*time.Second,
		"Interval in which we show update of the current file being generated.")
	timeout := flag.Duration(
		"timeout", 0,
		"Timeout duration to run the tool. Default 0 means no timeout.")
	flag.Parse()

	if *protoMessagePath == "" {
		fmt.Fprintln(os.Stderr,
			"Path of the proto message file (-msg-file param) should be specified "+
				"in the command line params.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	p, err := perf.NewPerfRunner(*projectId)
	if err != nil {
		log.Printf("Failed to create perf runner with err: %v", err)
		os.Exit(1)
	}

	log.Println("Creating Job configs...")
	errs := p.CreateConfigs(*protoMessagePath)
	if len(errs) != 0 {
		log.Printf("There are %d error(s) on creating job configs: ", len(errs))
		for _, err = range errs {
			log.Println(err)
		}
	}

	log.Println("Monitoring created jobs...")

	// Thread to poll for the status of perf run. This is useful to inform
	// the user of what's going on.
	ti := time.NewTicker(*updateInterval)
	go func() {
		for range ti.C {
			log.Println("Current Status:", p.GetStatus())
		}
	}()

	ctx := context.Background()
	if *timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, *timeout)
	}
	result, err := p.MonitorJobs(ctx)
	ti.Stop()
	if err != nil {
		log.Fatalf("Perf Job run failed with err: %v.", err)
	}

	log.Println("The perf run completed successfully. The run result:", *result)
}
