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
	"os"
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"../../dcp"
)

const (
	listProgressSubscription         string = "list_progress"
	copyProgressSubscription         string = "copy_progress"
	loadBigQueryProgressSubscription string = "loadbigquery_progress"

	listTopic   string = "list"
	copyTopic   string = "copy"
	loadBQTopic string = "loadbigquery"

	spannerInstance string = "spanner-instance"
	spannerDatabase string = "database"
)

func main() {
	// TODO(b/63103890): Do proper parsing of command line params.
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr,
			"Only Google cloud project id should be specified in the cmd line param.\n")
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
	loadBigQueryProgressSub := pubSubClient.Subscription(loadBigQueryProgressSubscription)

	listTopic := pubSubClient.Topic(listTopic)
	copyTopic := pubSubClient.Topic(copyTopic)
	loadBQTopic := pubSubClient.Topic(loadBQTopic)

	spannerClient, err := spanner.NewClient(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create spanner client, error %v.\n", err)
		os.Exit(1)
	}
	store := &dcp.SpannerStore{spannerClient}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not create storage client, error: %v.\n", err)
		os.Exit(1)
	}

	listingReceiver := dcp.MessageReceiver{
		Sub:   listProgressSub,
		Store: store,
		Handler: &dcp.ListProgressMessageHandler{
			Store:               store,
			ListingResultReader: &dcp.GCSListingResultReader{storageClient},
		},
	}

	uploadGCSReceiver := dcp.MessageReceiver{
		Sub:   copyProgressSub,
		Store: store,
		Handler: &dcp.UploadGCSProgressMessageHandler{
			Store: store,
		},
	}

	loadBigQueryReceiver := dcp.MessageReceiver{
		Sub:     loadBigQueryProgressSub,
		Store:   store,
		Handler: &dcp.LoadBQProgressMessageHandler{},
	}

	go listingReceiver.ReceiveMessages()
	go uploadGCSReceiver.ReceiveMessages()
	go loadBigQueryReceiver.ReceiveMessages()

	// Loop for infinity to queue tasks
	for {
		// TODO(b/63018200): The number of tasks to queue(100) should be
		// configurable, may be as a command line param.
		store.QueueTasks(100, listTopic, copyTopic, loadBQTopic)
		time.Sleep(10 * time.Millisecond)
	}
}
