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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/tests/perf"
)

const (
	defaultApiEndpoint = "https://api-dot-cloud-ingest-perf.appspot.com/"
)

func cleanupResources(skipCleanup bool, perfRunner *perf.PerfRunner) {
	if skipCleanup {
		log.Printf("No cleanup to perform; leaving any newly-created buckets intact.")
		return
	}

	log.Printf("Cleaning up any newly-created buckets.")

	// Fresh context, because the main one might have a timeout related to the actual test run.
	errs := perfRunner.CleanUp(context.Background())
	if len(errs) != 0 {
		log.Printf("There are %d error(s) on cleanup: ", len(errs))
		for _, err := range errs {
			log.Println(err)
		}
	}
}

func printAndEvaluateResults(validationResults []perf.ConfigValidationResult) bool {
	log.Printf("Validation Results: %+v\n", validationResults)

	// Pretty print results to stdout.
	success := true
	fmt.Println("Validation Results:")
	for _, result := range validationResults {
		fmt.Printf("Job %s: ", result.ConfigId)
		if len(result.Results) == 0 {
			fmt.Println("<no validation>")
		} else {
			fmt.Println()
			for _, v := range result.Results {
				fmt.Printf("- %s: ", v.Name)
				if v.Err != nil {
					fmt.Printf("ERROR: %v\n", v.Err)
					success = false
				} else if !v.Success {
					fmt.Printf("TEST FAILURE: %s\n", v.FailureMessage)
					success = false
				} else {
					fmt.Println("Success!")
				}
			}
		}
	}

	return success
}

func main() {
	protoMessagePath := flag.String("msg-file", "", "Path of the proto message file.")
	projectId := flag.String(
		"project-id", "",
		"Project id to run the perf. Empty project will choose the default project.")
	apiEndpoint := flag.String(
		"api-endpoint", defaultApiEndpoint,
		fmt.Sprintf("Webconsole backend API endpoint. The default is %s", defaultApiEndpoint))
	updateInterval := flag.Duration("update-interval", 5*time.Second,
		"Interval in which we show update of the current file being generated.")
	timeout := flag.Duration(
		"timeout", 0,
		"Timeout duration to run the tool. Default 0 means no timeout.")
	skipCleanup := flag.Bool(
		"skip-cleanup", false,
		"Set to true to skip resource cleanup (e.g. test bucket deletion). Defaults to false.")
	flag.Parse()

	if *protoMessagePath == "" {
		fmt.Fprintln(os.Stderr,
			"Path of the proto message file (-msg-file param) should be specified "+
				"in the command line params.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx := context.Background()

	// GCS Client Setup
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create storage client, error: %v.\n", err)
		os.Exit(1)
	}
	gcsClient := gcloud.NewGCSClient(storageClient)

	p, err := perf.NewPerfRunner(*projectId, *apiEndpoint, gcsClient)
	if err != nil {
		log.Printf("Failed to create perf runner with err: %v", err)
		os.Exit(1)
	}

	// Nesting this execution to make sure the deferred cleanup executes even when
	// we've failed a test and are about to fatally exit.
	result, validationResults, err := func() (*perf.PerfResult, []perf.ConfigValidationResult, error) {
		// Ensure we get things cleaned up before we exit.
		defer cleanupResources(*skipCleanup, p)

		log.Println("Creating Job configs...")
		errs := p.CreateConfigs(ctx, *protoMessagePath)
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

		if *timeout > 0 {
			ctx, _ = context.WithTimeout(ctx, *timeout)
		}
		result, err := p.MonitorJobs(ctx)
		ti.Stop()

		// Validate without re-using a context that's timing out.
		validationResult := p.ValidateResults(context.Background())

		return result, validationResult, err
	}()

	if err != nil {
		log.Fatalf("Perf Job run failed with err: %v.", err)
	}

	log.Println("Perf run completed, with run result:", *result)
	log.Printf("Validation Results: %+v\n", validationResults)
	success := printAndEvaluateResults(validationResults)

	// Unsuccessful exit code for any test failure/error.
	if !success {
		log.Fatal("Not all tests have completed successfully!")
	}
}
