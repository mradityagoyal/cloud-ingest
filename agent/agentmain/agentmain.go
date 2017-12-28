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
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent"
	"github.com/golang/glog"
	"google.golang.org/api/option"
)

const (
	listProgressTopic string = "cloud-ingest-list-progress"
	copyProgressTopic string = "cloud-ingest-copy-progress"

	listSubscription string = "cloud-ingest-list"
	copySubscription string = "cloud-ingest-copy"
)

var (
	projectID                string
	numberThreads            int
	maxPubSubLeaseExtenstion time.Duration
	credsFile                string
	chunkSize                int

	skipProcessListTasks bool
	skipProcessCopyTasks bool
)

func init() {
	flag.StringVar(&projectID, "projectid", "",
		"The Pub/Sub topics and subscriptions project id. Must be set!")
	flag.StringVar(&credsFile, "creds-file", "",
		"The service account JSON key file. Use the default credentials if empty.")
	flag.IntVar(&chunkSize, "chunk-size", 1<<25,
		"The resumable upload chuck size, default 32MB.")
	flag.IntVar(&numberThreads, "threads", 0,
		"The number of threads to process the copy tasks. If 0, will use the "+
			"default Pub/Sub client value (1000)")
	flag.DurationVar(&maxPubSubLeaseExtenstion, "pubsub-lease-extension", 0,
		"The max duration to extend the leases for a Pub/Sub message. If 0, will "+
			"use the default Pub/Sub client value (10 mins)")

	flag.BoolVar(&skipProcessListTasks, "skip-list", false,
		"Skip processing list tasks.")
	flag.BoolVar(&skipProcessCopyTasks, "skip-copy", false,
		"Skip processing copy tasks.")
	flag.Parse()
}

func main() {
	defer glog.Flush()
	ctx := context.Background()

	var pubSubErr, storageErr error
	var pubSubClient *pubsub.Client
	var storageClient *storage.Client

	if credsFile != "" {
		clientOptions := option.WithCredentialsFile(credsFile)
		pubSubClient, pubSubErr = pubsub.NewClient(ctx, projectID, clientOptions)
		storageClient, storageErr = storage.NewClient(ctx, clientOptions)
	} else {
		pubSubClient, pubSubErr = pubsub.NewClient(ctx, projectID)
		storageClient, storageErr = storage.NewClient(ctx)
	}

	if pubSubErr != nil {
		glog.Fatalf("Can not create Pub/Sub client, error: %+v.\n", pubSubErr)
	}

	if storageErr != nil {
		glog.Fatalf("Can not create storage client, error: %+v.\n", storageErr)
	}

	var wg sync.WaitGroup
	if !skipProcessListTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listSub := pubSubClient.Subscription(listSubscription)
			listSub.ReceiveSettings.MaxExtension = maxPubSubLeaseExtenstion
			listTopic := pubSubClient.Topic(listProgressTopic)

			listProcessor := agent.WorkProcessor{
				WorkSub:       listSub,
				ProgressTopic: listTopic,
				Handler:       agent.NewListHandler(storageClient, chunkSize),
			}
			listProcessor.Process(ctx)
		}()
	}

	if !skipProcessCopyTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copySub := pubSubClient.Subscription(copySubscription)
			copySub.ReceiveSettings.MaxExtension = maxPubSubLeaseExtenstion
			copySub.ReceiveSettings.MaxOutstandingMessages = numberThreads
			copyTopic := pubSubClient.Topic(copyProgressTopic)

			copyProcessor := agent.WorkProcessor{
				WorkSub:       copySub,
				ProgressTopic: copyTopic,
				Handler:       agent.NewCopyHandler(storageClient, chunkSize),
			}

			copyProcessor.Process(ctx)
		}()
	}

	wg.Wait()
}
