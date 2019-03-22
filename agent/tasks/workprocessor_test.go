package tasks

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/golang/protobuf/proto"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

type TestWorkHandler struct {
	responses map[string]*taskpb.TaskRespMsg
}

// Do handles the TaskReqMsg and returns a TaskRespMsg.
func (h *TestWorkHandler) Do(_ context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	return h.responses[taskReqMsg.TaskRelRsrcName]
}

// fakePubSubClient returns a fake pubsub client and a clean up function that should be called
// when the caller is finished with the returned client.
func fakePubSubClient(ctx context.Context, t *testing.T) (*pubsub.Client, func()) {
	t.Helper()
	server := pstest.NewServer()

	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("error connectioning to fake pub sub server %v", err)
	}

	projectID := "project"
	client, err := pubsub.NewClient(ctx, projectID, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("error creating fake pub sub client %v", err)
	}

	cleanUpFunc := func() {
		server.Close()
		conn.Close()
		client.Close()
	}
	return client, cleanUpFunc
}

func createTopic(ctx context.Context, t *testing.T, client *pubsub.Client, topicName string) *pubsub.Topic {
	t.Helper()
	topic, err := client.CreateTopic(ctx, topicName)
	if err != nil {
		t.Fatalf("error creating topic %v", err)
	}
	return topic
}

func createSubscription(ctx context.Context, t *testing.T, client *pubsub.Client, topic *pubsub.Topic, id string) *pubsub.Subscription {
	t.Helper()
	sub, err := client.CreateSubscription(ctx, id, pubsub.SubscriptionConfig{Topic: topic})
	if err != nil {
		t.Fatalf("error creating subscription %v", err)
	}
	return sub
}

// receiveMessages receives messages from the given pub sub subscription and writes them to the
// given channel.
func receiveMessages(ctx context.Context, msgs chan *pubsub.Message, sub *pubsub.Subscription) {
	receiveMsgs := func() {
		sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			msgs <- msg
		})
	}
	go receiveMsgs()
}

func getMessageOrTimeout(t *testing.T, msgs chan *pubsub.Message) *pubsub.Message {
	t.Helper()
	select {
	case message := <-msgs:
		return message
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting to receive message")
	}
	return nil
}

func TestWorkProcessorProcessMessage(t *testing.T) {
	ctx := context.Background()
	client, cleanUp := fakePubSubClient(ctx, t)
	defer cleanUp()

	progressTopic := createTopic(ctx, t, client, "progress")
	progressSub := createSubscription(ctx, t, client, progressTopic, "progressSub")

	workTopic := createTopic(ctx, t, client, "work")
	workSub := createSubscription(ctx, t, client, workTopic, "workSub")

	taskReqMsg := &taskpb.TaskReqMsg{
		TaskRelRsrcName:   "taskid",
		JobrunRelRsrcName: "jobrunid",
		JobRunVersion:     "0.0.0",
	}

	cm := &controlpb.Control{
		JobRunsBandwidths: []*controlpb.JobRunBandwidth{
			&controlpb.JobRunBandwidth{
				JobrunRelRsrcName: taskReqMsg.JobrunRelRsrcName,
				Bandwidth:         1,
			},
		},
	}
	rate.ProcessCtrlMsg(cm, nil)

	data, err := proto.Marshal(taskReqMsg)
	if err != nil {
		t.Fatalf("error marshalling task req message %v", err)
	}

	// Publish and receive task request message
	res := workTopic.Publish(ctx, &pubsub.Message{Data: data})
	res.Get(ctx)
	msgs := make(chan *pubsub.Message)
	receiveMessages(ctx, msgs, workSub)
	psTaskReqMsg := getMessageOrTimeout(t, msgs)

	want := &taskpb.TaskRespMsg{
		TaskRelRsrcName: taskReqMsg.TaskRelRsrcName,
		Status:          "SUCCESS",
	}
	wp := WorkProcessor{
		WorkSub:       workSub,
		ProgressTopic: progressTopic,
		Handlers: &HandlerRegistry{map[uint64]WorkHandler{
			0: &TestWorkHandler{map[string]*taskpb.TaskRespMsg{
				taskReqMsg.TaskRelRsrcName: want,
			}},
		}},
		StatsTracker: stats.NewTracker(ctx),
	}
	wp.processMessage(ctx, psTaskReqMsg)

	// Read and check task response message
	receiveMessages(ctx, msgs, progressSub)
	psTaskRespMsg := getMessageOrTimeout(t, msgs)

	var taskRespMsg taskpb.TaskRespMsg
	if err := proto.Unmarshal(psTaskRespMsg.Data, &taskRespMsg); err != nil {
		t.Fatalf("error decoding msg %s with error %v.", string(psTaskRespMsg.Data), err)
	}

	if taskRespMsg.TaskRelRsrcName != want.TaskRelRsrcName || taskRespMsg.Status != want.Status {
		t.Errorf("wp.processMessage(%v) = %v, want %v", taskReqMsg, taskRespMsg, want)
	}
}

func TestWorkProcessorProcessMessageNotActiveJob(t *testing.T) {
	ctx := context.Background()
	client, cleanUp := fakePubSubClient(ctx, t)
	defer cleanUp()

	progressTopic := createTopic(ctx, t, client, "progress")
	progressSub := createSubscription(ctx, t, client, progressTopic, "progressSub")

	workTopic := createTopic(ctx, t, client, "work")
	workSub := createSubscription(ctx, t, client, workTopic, "workSub")

	taskReqMsg := &taskpb.TaskReqMsg{
		TaskRelRsrcName:   "taskid2",
		JobrunRelRsrcName: "jobrunid2",
		JobRunVersion:     "0.0.0",
	}

	data, err := proto.Marshal(taskReqMsg)
	if err != nil {
		t.Fatalf("error marshalling task req message %v", err)
	}

	// Publish and receive task request message
	res := workTopic.Publish(ctx, &pubsub.Message{Data: data})
	res.Get(ctx)
	msgs := make(chan *pubsub.Message)
	receiveMessages(ctx, msgs, workSub)
	psTaskReqMsg := getMessageOrTimeout(t, msgs)

	wp := WorkProcessor{
		WorkSub:       workSub,
		ProgressTopic: progressTopic,
		Handlers:      nil,
		StatsTracker:  stats.NewTracker(ctx),
	}
	wp.processMessage(ctx, psTaskReqMsg)

	// Read and check task response message
	receiveMessages(ctx, msgs, progressSub)
	psTaskRespMsg := getMessageOrTimeout(t, msgs)

	var taskRespMsg taskpb.TaskRespMsg
	if err := proto.Unmarshal(psTaskRespMsg.Data, &taskRespMsg); err != nil {
		t.Fatalf("error decoding msg %s with error %v.", string(psTaskRespMsg.Data), err)
	}

	if taskReqMsg.TaskRelRsrcName != taskRespMsg.TaskRelRsrcName {
		t.Errorf("wp.processMessage(%v) TaskRelRsrcName = %v, want %v", taskReqMsg, taskRespMsg.TaskRelRsrcName, taskReqMsg.TaskRelRsrcName)
	}

	if taskRespMsg.Status != "FAILURE" {
		t.Errorf("wp.processMessage(%v) status = %v, want %v", taskReqMsg, taskRespMsg.Status, "FAILURE")
	}

	if taskRespMsg.FailureType != taskpb.FailureType_NOT_ACTIVE_JOBRUN {
		t.Errorf("wp.processMessage(%v) failure type = %v, want %v", taskReqMsg, taskRespMsg.FailureType, taskpb.FailureType_NOT_ACTIVE_JOBRUN)
	}
}

func TestWorkProcessorProcessMessageNoHandler(t *testing.T) {
	ctx := context.Background()
	client, cleanUp := fakePubSubClient(ctx, t)
	defer cleanUp()

	progressTopic := createTopic(ctx, t, client, "progress")
	progressSub := createSubscription(ctx, t, client, progressTopic, "progressSub")

	workTopic := createTopic(ctx, t, client, "work")
	workSub := createSubscription(ctx, t, client, workTopic, "workSub")

	taskReqMsg := &taskpb.TaskReqMsg{
		TaskRelRsrcName:   "taskid",
		JobrunRelRsrcName: "jobrunid",
		JobRunVersion:     "1.0.0",
	}

	cm := &controlpb.Control{
		JobRunsBandwidths: []*controlpb.JobRunBandwidth{
			&controlpb.JobRunBandwidth{
				JobrunRelRsrcName: taskReqMsg.JobrunRelRsrcName,
				Bandwidth:         1,
			},
		},
	}
	rate.ProcessCtrlMsg(cm, nil)

	data, err := proto.Marshal(taskReqMsg)
	if err != nil {
		t.Fatalf("error marshalling task req message %v", err)
	}

	// Publish and receive task request message
	res := workTopic.Publish(ctx, &pubsub.Message{Data: data})
	res.Get(ctx)
	msgs := make(chan *pubsub.Message)
	receiveMessages(ctx, msgs, workSub)
	psTaskReqMsg := getMessageOrTimeout(t, msgs)

	want := &taskpb.TaskRespMsg{
		TaskRelRsrcName: taskReqMsg.TaskRelRsrcName,
		Status:          "FAILURE",
		FailureType:     taskpb.FailureType_AGENT_UNSUPPORTED_VERSION,
	}
	wp := WorkProcessor{
		WorkSub:       workSub,
		ProgressTopic: progressTopic,
		Handlers: &HandlerRegistry{map[uint64]WorkHandler{
			0: &TestWorkHandler{map[string]*taskpb.TaskRespMsg{
				taskReqMsg.TaskRelRsrcName: want,
			}},
		}},
		StatsTracker: stats.NewTracker(ctx),
	}
	wp.processMessage(ctx, psTaskReqMsg)

	// Read and check task response message
	receiveMessages(ctx, msgs, progressSub)
	psTaskRespMsg := getMessageOrTimeout(t, msgs)

	var taskRespMsg taskpb.TaskRespMsg
	if err := proto.Unmarshal(psTaskRespMsg.Data, &taskRespMsg); err != nil {
		t.Fatalf("error decoding msg %s with error %v.", string(psTaskRespMsg.Data), err)
	}

	if taskRespMsg.TaskRelRsrcName != want.TaskRelRsrcName || taskRespMsg.Status != want.Status || taskRespMsg.FailureType != want.FailureType {
		t.Errorf("wp.processMessage(%v) = %v, want %v", taskReqMsg, taskRespMsg, want)
	}
}
