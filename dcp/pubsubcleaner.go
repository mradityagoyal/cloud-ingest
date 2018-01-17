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

package dcp

import (
	"context"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/glog"
)

// Cleaner scans the project table for unused topics and subscriptions.
// If a project has no associated job configs, it is assumed to be unused, and
// the cleaner will delete the associated topics and subscriptions.

const (
	unusedProjectsCleaningLimit int = 100
)

type PubSubCleaner struct {
	// TODO (b/71647771): PubSub Go client currently doesn't support an easy way to create a
	// subscription outside of the Client struct's project.  When
	// https://github.com/GoogleCloudPlatform/google-cloud-go/issues/849 is fixed,
	// drop the one-client-per-subscription.
	PubSubClientFunc func(ctx context.Context, projectID string) (gcloud.PS, error)
	Store            Store
}

func (p *PubSubCleaner) CleanPubSub() {
	unusedProjects, err := p.Store.GetUnusedProjects(unusedProjectsCleaningLimit)
	if err != nil {
		glog.Errorf("Could not get unused projects, error: %v.", err)
		return
	}
	for _, pi := range unusedProjects {
		pubSubClient, err := p.PubSubClientFunc(context.Background(), pi.ProjectID)
		if err != nil {
			glog.Errorf("Could not create PubSub client for unused project %s, error: %v.",
				pi.ProjectID, err)
			return
		}
		glog.Infof("Deleting topics and subscriptions for unused project %s", pi.ProjectID)
		listTopicGone := checkAndTryDeleteTopic(pubSubClient, pi.ListTopicID, pi.ProjectID)
		copyTopicGone := checkAndTryDeleteTopic(pubSubClient, pi.CopyTopicID, pi.ProjectID)
		listProgressSubAndTopicGone := checkAndTryDeleteSubAndParentTopic(
			pubSubClient, pi.ListProgressSubscriptionID, pi.ProjectID)
		copyProgressSubAndTopicGone := checkAndTryDeleteSubAndParentTopic(
			pubSubClient, pi.CopyProgressSubscriptionID, pi.ProjectID)

		if listTopicGone && copyTopicGone &&
			listProgressSubAndTopicGone && copyProgressSubAndTopicGone {
			// Remove unused project row.
			p.Store.DeleteProjectRow(pi.ProjectID)
		}
	}
}

// Tries to delete a topic, returns true if the topic is gone, false if it may
// still exist.
func checkAndTryDeleteTopic(client gcloud.PS, topicID, projectID string) bool {
	topic := client.Topic(topicID)
	topicExists, err := topic.Exists(context.Background())
	if err != nil {
		glog.Errorf("Could not check existence of topic %s for unused project %s, error: %v.",
			topicID, projectID, err)
		return false
	}
	if topicExists {
		if err = topic.Delete(context.Background()); err != nil {
			glog.Errorf("Could not delete topic %s for unused project %s, error: %v.",
				topicID, projectID, err)
			return false
		}
	}
	return true
}

// Tries to delete a subscription and the topic it is on. Returns true if the
// subscription and topic are gone, false if either may still exist.
// This function deletes the topic first, since the subscription is required to
// ascertain the topic.
func checkAndTryDeleteSubAndParentTopic(client gcloud.PS, subID, projectID string) bool {
	sub := client.Subscription(subID)
	subExists, err := sub.Exists(context.Background())
	if err != nil {
		glog.Errorf("Could not check existence of subscription %s for unused project %s, error: %v.",
			subID, projectID, err)
		return false
	}
	if subExists {
		subConfig, err := sub.Config(context.Background())
		if err != nil {
			glog.Errorf(
				"Could not retrieve config for project %s subscription %s, error: %v.",
				projectID, subID, err)
			return false
		}
		if !checkAndTryDeleteTopic(client, subConfig.Topic().ID(), projectID) {
			return false
		}
		err = sub.Delete(context.Background())
		if err != nil {
			glog.Errorf(
				"Could not delete subscription %s for unused project %s, error: %v.",
				subID, projectID, err)
			return false
		}
	}
	return true
}
