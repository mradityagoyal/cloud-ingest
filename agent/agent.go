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
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/control"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/copy"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/list"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/versions"
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
)

var (
	projectID                 = flag.String("projectid", "", "The Pub/Sub topics and subscriptions project id. Must be set!")
	numberThreads             = flag.Int("threads", 100, "The number of threads to process the copy tasks. If 0, will use the default Pub/Sub client value (1000).")
	numberConcurrentListTasks = flag.Int("number-concurrent-list-tasks", 4, "The maximum number of list tasks the agent will process at any given time.")
	maxPubSubLeaseExtenstion  = flag.Duration("pubsub-lease-extension", 0, "The max duration to extend the leases for a Pub/Sub message. If 0, will use the default Pub/Sub client value (10 mins).")
	credsFile                 = flag.String("creds-file", "", "The service account JSON key file. Use the default credentials if empty.")
	listTaskChunkSize         = flag.Int("list-task-chunk-size", 8*1024*1024, "The resumable upload chunk size used for list tasks, defaults to 8MiB.")

	pubsubPrefix = flag.String("pubsub-prefix", "", "Prefix of Pub/Sub topics and subscriptions names.")

	skipProcessListTasks = flag.Bool("skip-list", false, "Skip processing list tasks.")
	skipProcessCopyTasks = flag.Bool("skip-copy", false, "Skip processing copy tasks.")
	printVersion         = flag.Bool("version", false, "Print build/version info and exit.")

	enableStatsTracker = flag.Bool("enable-stats-log", true, "Enable stats logging to INFO logs.")

	listFileSizeThreshold          = flag.Int("list-file-size-threshold", 50000, "List tasks will keep listing directories until the number of listed files and directories exceeds this threshold, or until there are no more files/directories to list")
	maxMemoryForListingDirectories = flag.Int("max-memory-for-listing-directories", 20, "Maximum amount of memory agent will use in total (not per task) to store directories before writing them to a list file. Value is in MiB.")

	// Fields used to display version information. These defaults are
	// overridden when the release script builds in values through ldflags.
	buildVersion = "1.0.0"
	buildCommit  = "(local)"
	buildDate    = "Unknown; this is not an official release."
)

func init() {
	flag.Uint64Var(&glog.MaxSize, "max-log-size", 1<<28, "The maximum size of a log file in bytes, default 256MiB.")
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

	subID := fmt.Sprintf("%s%s-%d", *pubsubPrefix, controlSubscription, h.Sum64())
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

func createClients(ctx context.Context) (*pubsub.Client, *storage.Client, *http.Client) {
	clientOptions := make([]option.ClientOption, 0)
	if *credsFile != "" {
		clientOptions = append(clientOptions, option.WithCredentialsFile(*credsFile))
	}
	pubSubClient, err := pubsub.NewClient(ctx, *projectID, clientOptions...)
	if err != nil {
		glog.Fatalf("Couldn't create PubSub client, err: %v", err)
	}
	storageClient, err := storage.NewClient(ctx, clientOptions...)
	if err != nil {
		glog.Fatalf("Couldn't create Storage client, err: %v", err)
	}
	httpc, err := copy.NewResumableHttpClient(ctx, clientOptions...)
	if err != nil {
		glog.Fatalf("Couldn't create http.Client, err: %v", err)
	}
	return pubSubClient, storageClient, httpc
}

func main() {
	defer glog.Flush()
	ctx := context.Background()

	if *printVersion {
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

	pubSubClient, storageClient, httpc := createClients(ctx)

	var st *stats.Tracker
	if *enableStatsTracker {
		st = stats.NewTracker(ctx)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		pulseTopicWrapper := gcloud.NewPubSubTopicWrapper(pubSubClient.Topic(*pubsubPrefix + pulseTopic))
		// Wait for pulse topic to exist.
		if err := waitOnTopic(ctx, pulseTopicWrapper); err != nil {
			glog.Fatalf("Could not get PulseTopic: %s \n error: %v ", pulseTopicWrapper.ID(), err)
		}
		_, err := control.NewPulseSender(ctx, pulseTopicWrapper, logDir, st)
		if err != nil {
			glog.Fatalf("NewPulseSender(%v, %v) got err: %v ", pulseTopicWrapper, logDir, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		controlTopic := pubSubClient.Topic(*pubsubPrefix + controlTopic)
		controlTopicWrapper := gcloud.NewPubSubTopicWrapper(controlTopic)

		// Wait for control topic to exist.
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

	if !*skipProcessListTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listSub := pubSubClient.Subscription(*pubsubPrefix + listSubscription)
			listSub.ReceiveSettings.MaxExtension = *maxPubSubLeaseExtenstion
			listSub.ReceiveSettings.MaxOutstandingMessages = *numberConcurrentListTasks
			listSub.ReceiveSettings.Synchronous = true
			listTopic := pubSubClient.Topic(*pubsubPrefix + listProgressTopic)
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
			allowedDirBytes := *maxMemoryForListingDirectories * 1024 * 1024 / *numberConcurrentListTasks

			depthFirstListHandler := list.NewDepthFirstListHandler(storageClient, *listTaskChunkSize, *listFileSizeThreshold, allowedDirBytes)
			listProcessor := tasks.WorkProcessor{
				WorkSub:       listSub,
				ProgressTopic: listTopic,
				Handlers: tasks.NewHandlerRegistry(map[uint64]tasks.WorkHandler{
					0: list.NewListHandler(storageClient, *listTaskChunkSize),
					1: depthFirstListHandler,
					2: depthFirstListHandler,
				}),
				StatsTracker: st,
			}

			listProcessor.Process(ctx)
		}()
	}

	if !*skipProcessCopyTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copySub := pubSubClient.Subscription(*pubsubPrefix + copySubscription)
			copySub.ReceiveSettings.MaxExtension = *maxPubSubLeaseExtenstion
			copySub.ReceiveSettings.MaxOutstandingMessages = *numberThreads
			copySub.ReceiveSettings.Synchronous = true
			copyTopic := pubSubClient.Topic(*pubsubPrefix + copyProgressTopic)
			copyTopicWrapper := gcloud.NewPubSubTopicWrapper(copyTopic)

			// Wait for copy subscription to exist.
			if err := waitOnSubscription(ctx, copySub); err != nil {
				glog.Fatalf("Could not find copy subscription %s, error %+v", copySub.String(), err)
			}

			// Wait for copy topic to exist.
			if err := waitOnTopic(ctx, copyTopicWrapper); err != nil {
				glog.Fatalf("Could not find copy topic %s, error %+v", copyTopicWrapper.ID(), err)
			}

			copyHandler := copy.NewCopyHandler(storageClient, *numberThreads, httpc, st)
			copyProcessor := tasks.WorkProcessor{
				WorkSub:       copySub,
				ProgressTopic: copyTopic,
				Handlers: tasks.NewHandlerRegistry(map[uint64]tasks.WorkHandler{
					0: copyHandler,
					1: copyHandler,
					2: copyHandler,
				}),
				StatsTracker: st,
			}

			copyProcessor.Process(ctx)
		}()
	}

	wg.Wait()
}
