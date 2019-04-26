/*
Copyright 2019 Google Inc. All Rights Reserved.
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

package agentupdate

import (
	"bytes"
	"os"
	"testing"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

func TestProcessAgentUpdateMsg(t *testing.T) {
	agentID := &pulsepb.AgentId{
		HostName:  "host",
		ProcessId: "agent-0",
	}

	tests := []struct {
		desc             string
		auMsgs           []*controlpb.AgentUpdate
		wantUpdateSource []byte
	}{
		{
			desc: "Agent processes AgentUpdate message and creates the update source text file",
			auMsgs: []*controlpb.AgentUpdate{
				{
					AgentUpdateSources: []*controlpb.AgentUpdateSource{
						{
							AgentIds:  []*pulsepb.AgentId{agentID},
							UpdateUrl: "Test agent update source URL",
						},
					},
				},
			},
			wantUpdateSource: []byte("Test agent update source URL"),
		},
		{
			desc: "Agent update the text file",
			auMsgs: []*controlpb.AgentUpdate{
				{
					AgentUpdateSources: []*controlpb.AgentUpdateSource{
						{
							AgentIds:  []*pulsepb.AgentId{agentID},
							UpdateUrl: "Test agent update source URL",
						},
					},
				},
				{
					AgentUpdateSources: []*controlpb.AgentUpdateSource{
						{
							AgentIds:  []*pulsepb.AgentId{agentID},
							UpdateUrl: "Test update agent update source URL file",
						},
					},
				},
			},
			wantUpdateSource: []byte("Test update agent update source URL file"),
		},
	}

	agentLogsDir := "/tmp"
	for _, tc := range tests {
		for _, au := range tc.auMsgs {
			ProcessAgentUpdateMsg(au, agentID, agentLogsDir)
		}
		file, err := os.OpenFile(agentSourceFileName(agentLogsDir), os.O_RDONLY, 0755)
		if err != nil {
			t.Errorf("processAgentUpdateMsg('%s') failed to open agent source file, err: %v", tc.desc, err)
			continue
		}

		gotUpdateSource := make([]byte, len(tc.wantUpdateSource))
		_, err = file.Read(gotUpdateSource)
		if err != nil {
			t.Errorf("processAgentUpdateMsg('%s') failed to read agent source file, err: %v", tc.desc, err)
			continue
		}

		if !bytes.Equal(tc.wantUpdateSource, gotUpdateSource) {
			t.Errorf("processAgentUpdateMsg('%s') got %v, want %v", tc.desc, gotUpdateSource, tc.wantUpdateSource)
			continue
		}
		file.Close()
		os.Remove(agentSourceFileName(agentLogsDir))
	}
}

func TestProcessAgentUpdateMsgWithInvalidAgentID(t *testing.T) {
	agentID := &pulsepb.AgentId{
		HostName:  "host",
		ProcessId: "agent-0",
	}
	au := &controlpb.AgentUpdate{
		AgentUpdateSources: []*controlpb.AgentUpdateSource{
			{
				AgentIds: []*pulsepb.AgentId{
					{
						HostName:  "host",
						ProcessId: "agent-1",
					},
					{
						HostName:  "host",
						ProcessId: "agent-2",
					},
				},
				UpdateUrl: "Test update agent update source for invalid agent ID",
			},
		},
	}

	agentLogsDir := "../"
	ProcessAgentUpdateMsg(au, agentID, agentLogsDir)
	_, err := os.OpenFile(agentSourceFileName(agentLogsDir), os.O_RDONLY, 0755)
	if err == nil {
		t.Fatalf("ProcessAgentUpdateMsg succeeds with invalid agent IDs, want error")
	}
	if !os.IsNotExist(err) {
		t.Errorf("ProcessAgentUpdateMsg failed with err: %v, want IsNotExist error", err)
	}
}
