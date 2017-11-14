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

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
)

const (
	listProgressSubscription         string = "cloud-ingest-list-progress"
	copyProgressSubscription         string = "cloud-ingest-copy-progress"

	listTopic   string = "cloud-ingest-list"
	copyTopic   string = "cloud-ingest-copy"

	spannerInstance string = "cloud-ingest-spanner-instance"
	spannerDatabase string = "cloud-ingest-database"

	// The number of consecutive failures after which the program aborts
	maxNumFailures int = 15
	// The time for which the program sleeps between calls to store.QueueTasks
	queueTasksSleepTime time.Duration = 10 * time.Millisecond
	// The max time for which the program sleeps between calls to store.QueueTasks
	maxQueueTasksSleepTime time.Duration = 5 * time.Minute
)

// GetQueueTasksClosure returns a function that calls the function QueueTasks
// on the given store with the given values as the parameters.
func GetQueueTasksClosure(store *dcp.SpannerStore, num int,
	listTopic *pubsub.Topic, copyTopic *pubsub.Topic) func() error {

	return func() error {
		return store.QueueTasks(num, listTopic, copyTopic)
	}
}

func main() {
	// TODO(b/63103890): Do proper parsing of command line params.
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr,
			"Only Google cloud project id should be specified in the cmd line param.")
		os.Exit(1)
	}
	proj := os.Args[1]

	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		proj, spannerInstance, spannerDatabase)

	ctx := context.Background()

	pubSubClient, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create pubsub client, error: %v.\n", err)
		os.Exit(1)
	}
	listProgressSub := pubSubClient.Subscription(listProgressSubscription)
	copyProgressSub := pubSubClient.Subscription(copyProgressSubscription)

	listTopic := pubSubClient.Topic(listTopic)
	copyTopic := pubSubClient.Topic(copyTopic)

	spannerGCloudClient, err := spanner.NewClient(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create spanner client, error %v.\n", err)
		os.Exit(1)
	}
	spannerClient := dcp.NewSpannerClient(spannerGCloudClient)
	store := &dcp.SpannerStore{spannerClient}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create storage client, error: %v.\n", err)
		os.Exit(1)
	}
	gcsClient := dcp.NewGCSClient(storageClient)

	metadataReader := dcp.NewGCSObjectMetadataReader(gcsClient)
	listingReceiver := dcp.MessageReceiver{
		Sub:   listProgressSub,
		Store: store,
		Handler: &dcp.ListProgressMessageHandler{
			ListingResultReader: dcp.NewGCSListingResultReader(gcsClient),
			ObjectMetadataReader: metadataReader,
		},
	}

	uploadGCSReceiver := dcp.MessageReceiver{
		Sub:   copyProgressSub,
		Store: store,
		Handler: &dcp.UploadGCSProgressMessageHandler{
			ObjectMetadataReader: metadataReader,
		},
	}

	go listingReceiver.ReceiveMessages()
	go uploadGCSReceiver.ReceiveMessages()

	// The LogEntryProcessor will continuously export logs from the "LogEntries"
	// Spanner table to GCS.
	logEntryProcessor := dcp.LogEntryProcessor{gcsClient, store}
	logEntryProcessor.ProcessLogs()

	// Loop indefinitely to queue tasks.
	for {
		err := dcp.RetryWithExponentialBackoff(
			queueTasksSleepTime,
			maxQueueTasksSleepTime,
			maxNumFailures,
			"QueueTasks",
			// TODO(b/63018200): The number of tasks to queue(100) should be
			// configurable, maybe as a command line param.
			GetQueueTasksClosure(store, 100, listTopic, copyTopic),
		)
		if err != nil {
			log.Printf("Error in queueing tasks: %v.", err)
			os.Exit(1)
		}
		time.Sleep(queueTasksSleepTime)
	}
}
