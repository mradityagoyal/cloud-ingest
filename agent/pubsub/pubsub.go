package pubsub

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"runtime"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/list"
	"github.com/golang/glog"
)

const (
	listProgressTopicID   = "cloud-ingest-list-progress"
	copyProgressTopicID   = "cloud-ingest-copy-progress"
	deleteProgressTopicID = "cloud-ingest-delete-object-progress"
	pulseTopicID          = "cloud-ingest-pulse"
	controlTopicID        = "cloud-ingest-control"

	listSubscriptionID    = "cloud-ingest-list"
	copySubscriptionID    = "cloud-ingest-copy"
	deleteSubscriptionID  = "cloud-ingest-delete-object"
	controlSubscriptionID = "cloud-ingest-control"
)

var (
	pubsubPrefix             = flag.String("pubsub-prefix", "", "Prefix of Pub/Sub topics and subscriptions names.")
	maxPubSubLeaseExtenstion = flag.Duration("pubsub-lease-extension", 120*time.Minute, "The max duration to extend the leases for a Pub/Sub message. If 0, will use the default Pub/Sub client value (10 mins).")
	_                        = flag.Int("threads", 100, "This flag is no longer used and will be soon deprecated.")
	copyTasksPerCPU          = flag.Int("copy-tasks-per-cpu", 2, "Copy tasks to process (per CPU) in parallel. Can be overridden by setting copy-tasks.")
	copyTasks                = flag.Int("copy-tasks", 0, "Copy tasks to process in parallel. If > 0 this will override copy-tasks-per-cpu.")
	deleteTasks              = flag.Int("delete-tasks", 10, "Max delete tasks the agent will process at any given time. If 0, will use the default Pub/Sub client value (1000).")
)

// waitOnSubscription blocks until either the PubSub subscription exists, or returns an err.
func waitOnSubscription(ctx context.Context, sub *pubsub.Subscription) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			exists, err := sub.Exists(ctx)
			if err != nil {
				return err
			}
			if exists {
				fmt.Printf("PubSub subscription %q is ready.\n", sub.String())
				return nil
			}
			fmt.Printf("Waiting for PubSub subscription %q to exist.\n", sub.String())
			time.Sleep(10 * time.Second)
		}
	}
}

// waitOnTopic blocks until either the PubSub topic exists, or returns an err.
func waitOnTopic(ctx context.Context, topic *pubsub.Topic) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			exists, err := topic.Exists(ctx)
			if err != nil {
				return err
			}
			if exists {
				fmt.Printf("PubSub topic %q is ready.\n", topic.ID())
				return nil
			}
			fmt.Printf("Waiting for PubSub topic %q to exist.\n", topic.ID())
			time.Sleep(10 * time.Second)
		}
	}
}

func subscribeToControlTopic(ctx context.Context, client *pubsub.Client, topic *pubsub.Topic) (*pubsub.Subscription, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	h := fnv.New64a()
	h.Write([]byte(hostname))
	h.Write([]byte(fmt.Sprintf("%v", os.Getpid())))

	subID := fmt.Sprintf("%s%s-%d", *pubsubPrefix, controlSubscriptionID, h.Sum64())
	sub := client.Subscription(subID)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		glog.Infof("PubSub subscription %q already exists, probably another agent created it before.", sub.String())
		return sub, nil
	}
	return client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{Topic: topic})
}

// CreatePubSubTopicsAndSubs creates all of the PubSub topics and subs necessary for the Agent. If any of them can't
// be successfully created this function will glog.Fatal and kill the Agent.
//
// Where not overridden, the DefaultReceiveSettings are:
// ReceiveSettings{
//       MaxExtension:           10 * time.Minute,
//       MaxOutstandingMessages: 1000,
//       MaxOutstandingBytes:    1e9,
//       NumGoroutines:          1,
// }
// The default settings should be safe, because of the following reasons
// * MaxExtension:           DCP should not publish messages that are estimated to take more than 10 mins.
// * MaxOutstandingMessages: It's also capped by the memory, and this will speed up processing of small files.
// * MaxOutstandingBytes:    1GB memory should not be a problem for a modern machine.
// * NumGoroutines:          Does not need more than 1 routine to pull Pub/Sub messages.
func CreatePubSubTopicsAndSubs(ctx context.Context, pubSubClient *pubsub.Client) (listSub, copySub, controlSub, deleteSub *pubsub.Subscription, listTopic, copyTopic, pulseTopic, deleteTopic *pubsub.Topic) {
	var wg sync.WaitGroup
	wg.Add(8)
	go func() {
		defer wg.Done()
		listSub = pubSubClient.Subscription(*pubsubPrefix + listSubscriptionID)
		listSub.ReceiveSettings.MaxExtension = *maxPubSubLeaseExtenstion
		listSub.ReceiveSettings.MaxOutstandingMessages = *list.NumberConcurrentListTasks
		listSub.ReceiveSettings.Synchronous = true
		if err := waitOnSubscription(ctx, listSub); err != nil {
			glog.Fatalf("Could not find list subscription %s, error %+v", listSub.String(), err)
		}
	}()
	go func() {
		defer wg.Done()
		listTopic = pubSubClient.Topic(*pubsubPrefix + listProgressTopicID)
		if err := waitOnTopic(ctx, listTopic); err != nil {
			glog.Fatalf("Could not find list topic %s, error %+v", listTopic.ID(), err)
		}
	}()
	go func() {
		defer wg.Done()
		copySub = pubSubClient.Subscription(*pubsubPrefix + copySubscriptionID)
		copySub.ReceiveSettings.MaxExtension = *maxPubSubLeaseExtenstion
		ct := *copyTasks
		if ct <= 0 {
			ct = *copyTasksPerCPU * runtime.NumCPU()
		}
		glog.Info("CopySub MaxoutstandingMessages:", ct)
		copySub.ReceiveSettings.MaxOutstandingMessages = ct
		copySub.ReceiveSettings.Synchronous = true
		if err := waitOnSubscription(ctx, copySub); err != nil {
			glog.Fatalf("Could not find copy subscription %s, error %+v", copySub.String(), err)
		}
	}()
	go func() {
		defer wg.Done()
		copyTopic = pubSubClient.Topic(*pubsubPrefix + copyProgressTopicID)
		if err := waitOnTopic(ctx, copyTopic); err != nil {
			glog.Fatalf("Could not find copy topic %s, error %+v", copyTopic.ID(), err)
		}
	}()
	go func() {
		defer wg.Done()
		controlTopic := pubSubClient.Topic(*pubsubPrefix + controlTopicID)
		if err := waitOnTopic(ctx, controlTopic); err != nil {
			glog.Fatalf("Could not get ControlTopic: %s, got err: %v ", controlTopic.ID(), err)
		}
		var err error
		controlSub, err = subscribeToControlTopic(ctx, pubSubClient, controlTopic)
		if err != nil {
			glog.Fatalf("Could not create subscription to control topic %v, with err: %v", controlTopic.ID(), err)
		}
		controlSub.ReceiveSettings.MaxOutstandingMessages = 1
		if err := waitOnSubscription(ctx, controlSub); err != nil {
			glog.Fatalf("Could not find control subscription %s, error %+v", controlSub.String(), err)
		}
	}()
	go func() {
		defer wg.Done()
		pulseTopic = pubSubClient.Topic(*pubsubPrefix + pulseTopicID)
		if err := waitOnTopic(ctx, pulseTopic); err != nil {
			glog.Fatalf("Could not get PulseTopic: %s, got err: %v ", pulseTopic.ID(), err)
		}
	}()
	go func() {
		defer wg.Done()
		deleteSub = pubSubClient.Subscription(*pubsubPrefix + deleteSubscriptionID)
		deleteSub.ReceiveSettings.MaxExtension = *maxPubSubLeaseExtenstion
		deleteSub.ReceiveSettings.MaxOutstandingMessages = *deleteTasks
		deleteSub.ReceiveSettings.Synchronous = true
		if err := waitOnSubscription(ctx, deleteSub); err != nil {
			glog.Fatalf("Could not find delete subscription %s, error %+v", deleteSub.String(), err)
		}
	}()
	go func() {
		defer wg.Done()
		deleteTopic = pubSubClient.Topic(*pubsubPrefix + deleteProgressTopicID)
		if err := waitOnTopic(ctx, deleteTopic); err != nil {
			glog.Fatalf("Could not find delete topic %s, error %+v", deleteTopic.ID(), err)
		}
	}()
	wg.Wait()
	fmt.Println("All PubSub topics and subscriptions are ready.")

	return listSub, copySub, controlSub, deleteSub, listTopic, copyTopic, pulseTopic, deleteTopic
}
