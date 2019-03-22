package tasks

import (
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/versions"
	"github.com/golang/glog"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// HandlerRegistry manages handlers for all supported major job run versions.
type HandlerRegistry struct {
	handlers map[uint64]TaskHandler
}

// NewHandlerRegistry creates a new HandlerRegistry using the given major
// job run version to handler mappings. If the given map does not contain every supported major
// job run version, the function will perform a fatal log operation.
func NewHandlerRegistry(majorVersionToHandlers map[uint64]TaskHandler) *HandlerRegistry {
	handlers := make(map[uint64]TaskHandler)
	for v, h := range majorVersionToHandlers {
		handlers[v] = h
	}

	supportedVersions := versions.SupportedJobRuns()
	for _, v := range supportedVersions {
		if _, exists := handlers[v.Major]; !exists {
			glog.Fatalf("Lacking handler for supported major version %d", v.Major)
		}
	}

	return &HandlerRegistry{
		handlers: handlers,
	}
}

// HandlerForTaskReqMsg gets the appropriate handler for the given task request message. If the
// handler registry is unable to parse the job run version contained in the taskReqMsg or
// the registry does not contain the proper handler, an AgentError is returned.
func (h *HandlerRegistry) HandlerForTaskReqMsg(taskReqMsg *taskpb.TaskReqMsg) (TaskHandler, *common.AgentError) {
	jobRunVersion, err := versions.VersionFromString(taskReqMsg.JobRunVersion)
	if err != nil {
		glog.Errorf("Failed to parse job run version for task request message %v with err: %v", taskReqMsg, err)
		return nil, &common.AgentError{
			fmt.Sprintf("Failed to parse task request message job run version %v.", taskReqMsg.JobRunVersion),
			taskpb.FailureType_UNKNOWN_FAILURE,
		}
	}

	handler, exists := h.handlers[jobRunVersion.Major]
	if !exists {
		glog.Errorf("Handler does not exist for job run major version %d required for task request message %v", jobRunVersion.Major, taskReqMsg)
		return nil, &common.AgentError{
			fmt.Sprintf("Agent (version %v) does not support job run major version %v.", versions.AgentVersion(), jobRunVersion.Major),
			taskpb.FailureType_AGENT_UNSUPPORTED_VERSION,
		}
	}
	return handler, nil
}
