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
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

const (
	processListTopic string = "cloud-ingest-process-list"

	listProgressSubscription string = "cloud-ingest-list-progress"
	processListSubscription  string = "cloud-ingest-process-list"
	copyProgressSubscription string = "cloud-ingest-copy-progress"

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
	projectID            string
	tasksToQueue         int
	disableLogProcessing bool
)

func init() {
	flag.StringVar(&projectID, "projectid", "", "The project id to associate with this DCP. Must be set!")
	flag.IntVar(&tasksToQueue, "taskstoqueue", 100, "The number of tasks to queue at a time.")
	flag.BoolVar(&disableLogProcessing, "disablelogprocessing", false, "Disables writing logs to GCS.")
	flag.Parse()
	if projectID == "" {
		fmt.Println("The projectid flag must be set. Run 'dcpmain -h' for more info about flags.")
		os.Exit(1)
	}
}

// GetQueueTasksClosure returns a function that calls the function QueueTasks
// on the given store with the given values as the parameters.
func GetQueueTasksClosure(store *dcp.SpannerStore, num int,
	processListTopic gcloud.PSTopic, fallbackProjectID string) func() error {

	return func() error {
		return store.RoundRobinQueueTasks(num, processListTopic, fallbackProjectID)
	}
}

func main() {
	defer glog.Flush()
	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		projectID, spannerInstance, spannerDatabase)

	ctx := context.Background()

	pubSubGCloudClient, err := pubsub.NewClient(ctx, projectID)
	pubSubClient := gcloud.NewPubSubClient(pubSubGCloudClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create pubsub client, error: %v.\n", err)
		os.Exit(1)
	}
	listProgressSub := pubSubClient.Subscription(listProgressSubscription)
	processListSub := pubSubClient.Subscription(processListSubscription)
	copyProgressSub := pubSubClient.Subscription(copyProgressSubscription)

	processListTopic := pubSubClient.Topic(processListTopic)

	spannerGCloudClient, err := spanner.NewClient(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create spanner client, error %v.\n", err)
		os.Exit(1)
	}
	spannerClient := gcloud.NewSpannerClient(spannerGCloudClient)
	store := &dcp.SpannerStore{spannerClient, pubSubClient}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create storage client, error: %v.\n", err)
		os.Exit(1)
	}
	gcsClient := gcloud.NewGCSClient(storageClient)

	metadataReader := dcp.NewGCSObjectMetadataReader(gcsClient)
	listReceiver := dcp.MessageReceiver{
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
			ListingResultReader: dcp.NewGCSListingResultReader(gcsClient),
		},
	}

	copyReceiver := dcp.MessageReceiver{
		Sub:   copyProgressSub,
		Store: store,
		Handler: &dcp.CopyProgressMessageHandler{
			ObjectMetadataReader: metadataReader,
		},
	}

	go listReceiver.ReceiveMessages(ctx)
	go processListReceiver.ReceiveMessages(ctx)
	go copyReceiver.ReceiveMessages(ctx)

	// The LogEntryProcessor will continuously export logs from the "LogEntries"
	// Spanner table to GCS.
	if !disableLogProcessing {
		logEntryProcessor := dcp.LogEntryProcessor{gcsClient, store}
		logEntryProcessor.ProcessLogs()
	}

	// Loop indefinitely to queue tasks.
	for {
		err := helpers.RetryWithExponentialBackoff(
			queueTasksSleepTime,
			maxQueueTasksSleepTime,
			maxNumFailures,
			"QueueTasks",
			GetQueueTasksClosure(store, tasksToQueue, processListTopic, projectID),
		)
		if err != nil {
			glog.Fatalf("Error in queueing tasks: %v.", err)
		}
		time.Sleep(queueTasksSleepTime)
	}
}
