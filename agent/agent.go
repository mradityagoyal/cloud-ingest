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
	"os/signal"
	"syscall"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/control"
	pubsubinternal "github.com/GoogleCloudPlatform/cloud-ingest/agent/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/copy"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/versions"
	"github.com/golang/glog"
	"google.golang.org/api/option"
)

var (
	projectID          = flag.String("projectid", "", "The Pub/Sub topics and subscriptions project id. Must be set!")
	credsFile          = flag.String("creds-file", "", "The service account JSON key file. Use the default credentials if empty.")
	printVersion       = flag.Bool("version", false, "Print build/version info and exit.")
	enableStatsTracker = flag.Bool("enable-stats-log", true, "Enable stats logging to INFO logs.")

	// Fields used to display version information. These defaults are
	// overridden when the release script builds in values through ldflags.
	buildVersion = "1.0.0"
	buildCommit  = "(local)"
	buildDate    = "Unknown; this is not an official release."
)

func init() {
	flag.Uint64Var(&glog.MaxSize, "max-log-size", 1<<28, "The maximum size of a log file in bytes, default 256MiB.")
	flag.Parse()

	if len(*projectID) == 0 {
		glog.Fatalf("flag projectid must be set")
	}
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

func catchCtrlC(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range c {
			fmt.Println("\n\nCaught ^C, cleaning up and exiting (please wait)...")
			cancel() // Cancel the main context.
			// Further CTRL-Cs will be treated normally (forcing immediate exit).
			signal.Reset()
		}
	}()
}

func main() {
	defer fmt.Println("Exited gracefully.")
	defer glog.Flush()
	ctx, cancel := context.WithCancel(context.Background())
	catchCtrlC(cancel)

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

	// Create the PubSub topics and subscriptions.
	listSub, copySub, controlSub, listTopic, copyTopic, pulseTopic := pubsubinternal.CreatePubSubTopicsAndSubs(ctx, pubSubClient)
	defer controlSub.Delete(context.Background())
	var st *stats.Tracker
	if *enableStatsTracker {
		st = stats.NewTracker(ctx) // Created after PubSub topics/subs so STDOUT doesn't get stomped.
	}

	control.NewPulseSender(ctx, pubsubinternal.NewPubSubTopicWrapper(pulseTopic), logDir, st)

	controlHandler := control.NewControlHandler(controlSub, st)
	go controlHandler.Process(ctx)

	listProcessor := tasks.NewListProcessor(storageClient, listSub, listTopic, st)
	go listProcessor.Process(ctx)

	copyProcessor := tasks.NewCopyProcessor(storageClient, httpc, copySub, copyTopic, st)
	go copyProcessor.Process(ctx)

	// Block until the ctx is cancelled.
	select {
	case <-ctx.Done():
		break
	}
}
