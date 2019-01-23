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
	"hash/fnv"
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/control"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/versions"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/glog"
	"google.golang.org/api/option"
)

const (
	listProgressTopic = "cloud-ingest-list-progress"
	copyProgressTopic = "cloud-ingest-copy-progress"
	pulseTopic        = "cloud-ingest-pulse"
	controlTopic      = "cloud-ingest-control"

	listSubscription    = "cloud-ingest-list"
	copySubscription    = "cloud-ingest-copy"
	controlSubscription = "cloud-ingest-control"

	// copyOutstandingMsgsFactor is the multiplication factor to calculate the max number
	// of outstanding copy messages from the max number of concurrent copy
	// operations. Bassically, this keeps more messages in a buffer for performance.
	copyOutstandingMsgsFactor = 1
)

var (
	projectID                 string
	numberThreads             int
	numberConcurrentListTasks int
	maxPubSubLeaseExtenstion  time.Duration
	credsFile                 string
	listTaskChunkSize         int

	pubsubPrefix string

	skipProcessListTasks bool
	skipProcessCopyTasks bool
	printVersion         bool

	enableStatsTracker bool

	listFileSizeThreshold          int
	maxMemoryForListingDirectories int

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
	flag.IntVar(&listTaskChunkSize, "list-task-chunk-size", 8*1024*1024,
		"The resumable upload chunk size used for list tasks, defaults to 8MiB.")
	flag.IntVar(&numberThreads, "threads", 100,
		"The number of threads to process the copy tasks. If 0, will use the "+
			"default Pub/Sub client value (1000).")
	flag.IntVar(&numberConcurrentListTasks, "number-concurrent-list-tasks", 4,
		"The maximum number of list tasks the agent will process at any given time.")
	flag.DurationVar(&maxPubSubLeaseExtenstion, "pubsub-lease-extension", 0,
		"The max duration to extend the leases for a Pub/Sub message. If 0, will "+
			"use the default Pub/Sub client value (10 mins).")
	flag.Uint64Var(&glog.MaxSize, "max-log-size", 1<<28,
		"The maximum size of a log file in bytes, default 268435456 bytes(256MB).")

	flag.BoolVar(&skipProcessListTasks, "skip-list", false,
		"Skip processing list tasks.")
	flag.BoolVar(&skipProcessCopyTasks, "skip-copy", false,
		"Skip processing copy tasks.")
	flag.BoolVar(&printVersion, "version", false,
		"Print build/version info and exit.")

	flag.StringVar(&pubsubPrefix, "pubsub-prefix", "",
		"Prefix of Pub/Sub topics and subscriptions names.")

	flag.BoolVar(&enableStatsTracker, "enable-stats-log", true, "Enable stats logging to INFO logs.")

	flag.IntVar(&listFileSizeThreshold, "list-file-size-threshold", 10000,
		"List tasks will keep listing directories until the number of listed files and directories exceeds this threshold, or until there are no more files/directories to list")
	flag.IntVar(&maxMemoryForListingDirectories, "max-memory-for-listing-directories", 20,
		"Maximum amount of memory agent will use in total (not per task) to store directories before writing them to a list file. Value is in MiB.")

	flag.Parse()
}

func printVersionInfo() {
	fmt.Printf("Google Cloud Ingest Agent %s\n", buildVersion)
	fmt.Printf("Git Commit: %s\nBuild Date: %s\n", buildCommit, buildDate)
}

// createLogDirIfNeeded returns the value of the directory that glog logs will
// be written, creating that directory if it does not already exist. It returns
// an error if the directory is invalid or could not be created.
func createLogDirIfNeeded() (string, error) {
	if f := flag.Lookup("log_dir"); f != nil && f.Value.String() != "" {
		logDir := f.Value.String()
		fd, err := os.Stat(logDir)

		if os.IsNotExist(err) {
			if err2 := os.MkdirAll(logDir, 0777); err2 != nil {
				return logDir, err2
			}
		} else if err != nil {
			return logDir, err
		} else if fd.Mode().IsRegular() {
			return logDir, fmt.Errorf("log dir %s is a file", logDir)
		}
		return logDir, nil
	}

	// This is coupled with glog's default value since glog
	// does not provide a way to look up the default.
	return os.TempDir(), nil
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

// waitOnTopic blocks until either the passed-in PSTopic exists, or
// an error occurs (including context end). In all cases where we return without
// the PSTopic existing, we return the relevant error.
func waitOnTopic(ctx context.Context, topic gcloud.PSTopic) error {
	exists, err := topic.Exists(ctx)
	if err != nil {
		return err
	}

	t := time.NewTicker(10 * time.Second)
	for !exists {
		fmt.Printf("Waiting for topic %s to exist.\n", topic.ID())

		select {
		case <-t.C:
			exists, err = topic.Exists(ctx)
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
	fmt.Printf("Topic %s is ready!\n", topic.ID())
	return nil
}

func subscribeToControlTopic(ctx context.Context, client *pubsub.Client, topic *pubsub.Topic) (*pubsub.Subscription, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	h := fnv.New64a()
	h.Write([]byte(hostname))
	h.Write([]byte(fmt.Sprintf("%v", os.Getpid())))

	subID := fmt.Sprintf("%s%s-%d", pubsubPrefix, controlSubscription, h.Sum64())
	sub := client.Subscription(subID)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		glog.Infof("Subscription %s already exists, probably another agent created it before.", sub.String())
		return sub, nil
	}
	return client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{Topic: topic})
}

func main() {
	defer glog.Flush()
	ctx := context.Background()

	if printVersion {
		printVersionInfo()
		os.Exit(0)
	}
	err := versions.SetAgentVersion(buildVersion)
	if err != nil {
		glog.Fatalf("failed to set agent version with error %v", err)
	}

	logDir, err := createLogDirIfNeeded()
	if err != nil {
		glog.Fatalf("error accessing log output dir %s: %v\n", logDir, err)
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		pulseTopicWrapper := gcloud.NewPubSubTopicWrapper(pubSubClient.Topic(pubsubPrefix + pulseTopic))
		// Wait for pulse topic to exist.
		if err := waitOnTopic(ctx, pulseTopicWrapper); err != nil {
			glog.Fatalf("Could not get PulseTopic: %s \n error: %v ", pulseTopicWrapper.ID(), err)
		}
		_, err := control.NewPulseSender(ctx, pulseTopicWrapper, logDir)
		if err != nil {
			glog.Fatalf("NewPulseSender(%v, %v) got err: %v ", pulseTopicWrapper, logDir, err)
		}
	}()

	var st *stats.Tracker
	if enableStatsTracker {
		st = stats.NewTracker(ctx)
	}

	var listProcessor, copyProcessor agent.WorkProcessor

	if !skipProcessListTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listSub := pubSubClient.Subscription(pubsubPrefix + listSubscription)
			listSub.ReceiveSettings.MaxExtension = maxPubSubLeaseExtenstion
			listSub.ReceiveSettings.MaxOutstandingMessages = numberConcurrentListTasks
			listSub.ReceiveSettings.Synchronous = true
			listTopic := pubSubClient.Topic(pubsubPrefix + listProgressTopic)
			listTopicWrapper := gcloud.NewPubSubTopicWrapper(listTopic)

			// Wait for list subscription to exist.
			if err := waitOnSubscription(ctx, listSub); err != nil {
				glog.Fatalf("Could not find list subscription %s, error %+v", listSub.String(), err)
			}

			// Wait for list topic to exist.
			if err := waitOnTopic(ctx, listTopicWrapper); err != nil {
				glog.Fatalf("Could not find list topic %s, error %+v", listTopicWrapper.ID(), err)
			}

			// Convert maxMemoryForListingDirectories to bytes and divide it equally between
			// the list task processing threads.
			allowedDirBytes := maxMemoryForListingDirectories * 1024 * 1024 / numberConcurrentListTasks

			depthFirstListHandler := agent.NewDepthFirstListHandler(storageClient, listTaskChunkSize, listFileSizeThreshold, allowedDirBytes)
			listProcessor = agent.WorkProcessor{
				WorkSub:       listSub,
				ProgressTopic: listTopic,
				Handlers: agent.NewHandlerRegistry(map[uint64]agent.WorkHandler{
					0: agent.NewListHandler(storageClient, listTaskChunkSize),
					1: depthFirstListHandler,
					2: depthFirstListHandler,
				}),
				StatsTracker: st,
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
			copySub.ReceiveSettings.MaxOutstandingMessages = numberThreads * copyOutstandingMsgsFactor
			copySub.ReceiveSettings.Synchronous = true
			copyTopic := pubSubClient.Topic(pubsubPrefix + copyProgressTopic)
			copyTopicWrapper := gcloud.NewPubSubTopicWrapper(copyTopic)

			// Wait for copy subscription to exist.
			if err := waitOnSubscription(ctx, copySub); err != nil {
				glog.Fatalf("Could not find copy subscription %s, error %+v", copySub.String(), err)
			}

			// Wait for copy topic to exist.
			if err := waitOnTopic(ctx, copyTopicWrapper); err != nil {
				glog.Fatalf("Could not find copy topic %s, error %+v", copyTopicWrapper.ID(), err)
			}

			copyHandler := agent.NewCopyHandler(storageClient, numberThreads, httpc, st)
			copyProcessor = agent.WorkProcessor{
				WorkSub:       copySub,
				ProgressTopic: copyTopic,
				Handlers: agent.NewHandlerRegistry(map[uint64]agent.WorkHandler{
					0: copyHandler,
					1: copyHandler,
					2: copyHandler,
				}),
				StatsTracker: st,
			}

			copyProcessor.Process(ctx)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		controlTopic := pubSubClient.Topic(pubsubPrefix + controlTopic)
		controlTopicWrapper := gcloud.NewPubSubTopicWrapper(controlTopic)

		// Wait for copy topic to exist.
		if err := waitOnTopic(ctx, controlTopicWrapper); err != nil {
			glog.Fatalf("Could not find control topic %s, error %+v", controlTopicWrapper.ID(), err)
		}

		controlSub, err := subscribeToControlTopic(ctx, pubSubClient, controlTopic)
		if err != nil {
			glog.Fatalf("Could not create subscription to control topic %v, with err: %v", controlTopic, err)
		}

		ch := control.NewControlHandler(controlSub, st)
		if err := ch.HandleControlMessages(ctx); err != nil {
			glog.Fatalf("Failed handling control messages with err: %v.", err)
		}
	}()

	wg.Wait()
}
