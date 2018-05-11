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
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent"
	"github.com/golang/glog"
	"google.golang.org/api/option"
)

const (
	listProgressTopic = "cloud-ingest-list-progress"
	copyProgressTopic = "cloud-ingest-copy-progress"

	listSubscription = "cloud-ingest-list"
	copySubscription = "cloud-ingest-copy"
)

var (
	projectID                string
	numberThreads            int
	maxPubSubLeaseExtenstion time.Duration
	credsFile                string
	chunkSize                int

	pubsubPrefix string

	skipProcessListTasks bool
	skipProcessCopyTasks bool
	printVersion         bool

	// Fields used to display version information. These defaults are
	// overridden when the release script builds in values through ldflags.
	buildVersion = "v0.0.0-development"
	buildCommit  = "(local)"
	buildDate    = "Unknown; this is not an official release."
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
	flag.BoolVar(&printVersion, "version", false,
		"Print build/version info and exit.")

	flag.StringVar(&pubsubPrefix, "pubsub-prefix", "",
		"Prefix of Pub/Sub topics and subscriptions names.")

	flag.Parse()
}

func printVersionInfo() {
	fmt.Printf("Google Cloud Ingest Agent %s\n", buildVersion)
	fmt.Printf("Git Commit: %s\nBuild Date: %s\n", buildCommit, buildDate)
}

// waitOnSubscription blocks until either the passed-in subscription exists, or
// an error occurs (including context end). In all cases where we return without
// the subscription existing, we return the relevant error.
func waitOnSubscription(ctx context.Context, sub *pubsub.Subscription) error {
	exists, err := sub.Exists(ctx)
	if err != nil {
		return err
	}

	t := time.NewTicker(10 * time.Second)

	for !exists {
		fmt.Printf("Waiting for subscription %s to exist. If this is the first run for this project, "+
			"create your first transfer job.\n", sub.String())

		select {
		case <-t.C:
			exists, err = sub.Exists(ctx)
			if err != nil {
				t.Stop()
				return err
			}
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		}
	}

	t.Stop()
	fmt.Printf("Subscription %s is ready!\n", sub.String())
	return nil
}

func main() {
	defer glog.Flush()
	ctx := context.Background()

	if printVersion {
		printVersionInfo()
		os.Exit(0)
	}

	var pubSubErr, storageErr, httpcErr error
	var pubSubClient *pubsub.Client
	var storageClient *storage.Client
	var httpc *http.Client

	if credsFile != "" {
		clientOptions := option.WithCredentialsFile(credsFile)
		pubSubClient, pubSubErr = pubsub.NewClient(ctx, projectID, clientOptions)
		storageClient, storageErr = storage.NewClient(ctx, clientOptions)
		httpc, httpcErr = agent.NewResumableHttpClient(ctx, clientOptions)
	} else {
		pubSubClient, pubSubErr = pubsub.NewClient(ctx, projectID)
		storageClient, storageErr = storage.NewClient(ctx)
		httpc, httpcErr = agent.NewResumableHttpClient(ctx)
	}

	if pubSubErr != nil {
		glog.Fatalf("Can't create Pub/Sub client, error: %+v\n", pubSubErr)
	}
	if storageErr != nil {
		glog.Fatalf("Can't create storage client, error: %+v\n", storageErr)
	}
	if httpcErr != nil {
		glog.Fatalf("Can't create http.Client, error: %+v\n", httpcErr)
	}

	var wg sync.WaitGroup
	if !skipProcessListTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listSub := pubSubClient.Subscription(pubsubPrefix + listSubscription)
			listSub.ReceiveSettings.MaxExtension = maxPubSubLeaseExtenstion
			listTopic := pubSubClient.Topic(pubsubPrefix + listProgressTopic)

			// Wait for list subscription to exist.
			if err := waitOnSubscription(ctx, listSub); err != nil {
				glog.Fatalf("Could not find list subscription %s, error %+v", listSub.String(), err)
			}

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
			copySub := pubSubClient.Subscription(pubsubPrefix + copySubscription)
			copySub.ReceiveSettings.MaxExtension = maxPubSubLeaseExtenstion
			copySub.ReceiveSettings.MaxOutstandingMessages = numberThreads
			copyTopic := pubSubClient.Topic(pubsubPrefix + copyProgressTopic)

			// Wait for copy subscription to exist.
			if err := waitOnSubscription(ctx, copySub); err != nil {
				glog.Fatalf("Could not find copy subscription %s, error %+v", copySub.String(), err)
			}

			copyProcessor := agent.WorkProcessor{
				WorkSub:       copySub,
				ProgressTopic: copyTopic,
				Handler:       agent.NewCopyHandler(storageClient, chunkSize, httpc),
			}

			copyProcessor.Process(ctx)
		}()
	}

	wg.Wait()
}
