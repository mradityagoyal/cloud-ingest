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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
)

const (
	listProgressSubscription string = "cloud-ingest-list-progress"
	processListSubscription string = "cloud-ingest-process-list"
	copyProgressSubscription string = "cloud-ingest-copy-progress"

	listTopic string = "cloud-ingest-list"
	processListTopic string = "cloud-ingest-process-list"
	copyTopic string = "cloud-ingest-copy"

	spannerInstance string = "cloud-ingest-spanner-instance"
	spannerDatabase string = "cloud-ingest-database"

	// The number of consecutive failures after which the program aborts
	maxNumFailures int = 15
	// The time for which the program sleeps between calls to store.QueueTasks
	queueTasksSleepTime time.Duration = 1 * time.Second
	// The max time for which the program sleeps between calls to store.QueueTasks
	maxQueueTasksSleepTime time.Duration = 5 * time.Minute
)

var (
	projectID    string
	tasksToQueue int
)

func init() {
	flag.StringVar(&projectID, "projectid", "", "The project id to associate with this DCP. Must be set!")
	flag.IntVar(&tasksToQueue, "taskstoqueue", 100, "The number of tasks to queue at a time.")
	flag.Parse()
	if projectID == "" {
		fmt.Println("The projectid flag must be set. Run 'dcpmain -h' for more info about flags.")
		os.Exit(1)
	}
}

// GetQueueTasksClosure returns a function that calls the function QueueTasks
// on the given store with the given values as the parameters.
func GetQueueTasksClosure(store *dcp.SpannerStore, num int,
	listTopic, processListTopic, copyTopic *pubsub.Topic) func() error {

	return func() error {
		return store.QueueTasks(num, listTopic, processListTopic, copyTopic)
	}
}

func main() {
	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		projectID, spannerInstance, spannerDatabase)

	ctx := context.Background()

	pubSubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create pubsub client, error: %v.\n", err)
		os.Exit(1)
	}
	listProgressSub := pubSubClient.Subscription(listProgressSubscription)
	processListSub := pubSubClient.Subscription(processListSubscription)
	copyProgressSub := pubSubClient.Subscription(copyProgressSubscription)

	listTopic := pubSubClient.Topic(listTopic)
	processListTopic := pubSubClient.Topic(processListTopic)
	copyTopic := pubSubClient.Topic(copyTopic)

	spannerGCloudClient, err := spanner.NewClient(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create spanner client, error %v.\n", err)
		os.Exit(1)
	}
	spannerClient := gcloud.NewSpannerClient(spannerGCloudClient)
	store := &dcp.SpannerStore{spannerClient}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create storage client, error: %v.\n", err)
		os.Exit(1)
	}
	gcsClient := gcloud.NewGCSClient(storageClient)

	metadataReader := dcp.NewGCSObjectMetadataReader(gcsClient)
	listingReceiver := dcp.MessageReceiver{
		Sub:   listProgressSub,
		Store: store,
		Handler: &dcp.ListProgressMessageHandler{
			ObjectMetadataReader: metadataReader,
		},
	}

	processListReceiver := dcp.MessageReceiver{
		Sub:   processListSub,
		Store: store,
		Handler: &dcp.ProcessListMessageHandler{
			ListingResultReader:  dcp.NewGCSListingResultReader(gcsClient),
		},
	}

	uploadGCSReceiver := dcp.MessageReceiver{
		Sub:   copyProgressSub,
		Store: store,
		Handler: &dcp.UploadGCSProgressMessageHandler{
			ObjectMetadataReader: metadataReader,
		},
	}

	go listingReceiver.ReceiveMessages(ctx)
	go processListReceiver.ReceiveMessages(ctx)
	go uploadGCSReceiver.ReceiveMessages(ctx)

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
			GetQueueTasksClosure(store, tasksToQueue, listTopic, processListTopic, copyTopic),
		)
		if err != nil {
			log.Printf("Error in queueing tasks: %v.", err)
			os.Exit(1)
		}
		time.Sleep(queueTasksSleepTime)
	}
}
