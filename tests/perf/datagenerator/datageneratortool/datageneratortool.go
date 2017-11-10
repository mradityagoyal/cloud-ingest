/* Copyright 2017 Google Inc. All Rights Reserved.
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

// Datagenerator tool is a binary to generate the load testing data.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/datagenerator"
)

func main() {
	protoMessagePath := flag.String("msg-file", "", "Path of the proto message file.")
	updateInterval := flag.Duration("update-interval", 5*time.Second,
		"Interval in which we show update of the current file being generated. 0 for no updates")
	flag.Parse()

	if *protoMessagePath == "" {
		fmt.Fprintln(os.Stderr,
			"Path of the proto message file should be specified in the command line params.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	g, err := datagenerator.NewGeneratorFromProtoFile(*protoMessagePath)
	if err != nil {
		log.Printf("Error constructing generator with err: %v", err)
		os.Exit(1)
	}

	// Thread to poll for the status of the generation. This is useful to inform
	// the user of what's going on.
	var ti *time.Ticker
	if *updateInterval > 0 {
		ti = time.NewTicker(*updateInterval)
		go func() {
			for range ti.C {
				log.Println(g.GetStatus())
			}
		}()
	}

	stTime := time.Now()
	errs := g.GenerateObjects()
	if ti != nil {
		ti.Stop()
	}
	timeElapsed := time.Now().Sub(stTime)
	log.Printf("Generation took %v.", timeElapsed)

	if len(errs) > 0 {
		log.Println("The list of errors resulted in the generation:")
		for _, err := range errs {
			log.Printf(err.Error())
		}
	}
}
