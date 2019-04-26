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
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

func agentSourceFileName(agentLogsDir string) string {
	return fmt.Sprintf("%s/agent_source_%v.txt", agentLogsDir, os.Getpid())
}

// ProcessAgentUpdateMsg processes AgentUpdate message in Control message
func ProcessAgentUpdateMsg(au *controlpb.AgentUpdate, agentID *pulsepb.AgentId, agentLogsDir string) {
	if au == nil {
		return
	}

	agentUpdateSource := agentUpdateUrl(agentID, au)

	fn := agentSourceFileName(agentLogsDir)
	// Delete the text file if AgentUpdate message does not contain the current agent ID,
	// the update script will detect this and automatically switch back to stable version
	if len(agentUpdateSource) == 0 {
		err := os.Remove(fn)
		if err != nil && !os.IsNotExist(err) {
			glog.Errorf("Failed to remove file %s, err: %v", fn, err)
		}
		return
	}

	file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		glog.Errorf("Failed to open/create file: %s, err: %v", fn, err)
		return
	}
	defer file.Close()

	// Rewrite the file with new update URL
	_, err = file.WriteString(agentUpdateSource)
	if err != nil {
		glog.Errorf("Failed to write agent update source into file, got err: %v", err)
		return
	}
}

func agentUpdateUrl(agentID *pulsepb.AgentId, au *controlpb.AgentUpdate) string {
	for _, source := range au.GetAgentUpdateSources() {
		agentIDs := source.GetAgentIds()
		for _, id := range agentIDs {
			if proto.Equal(id, agentID) {
				return source.GetUpdateUrl()
			}
		}
	}
	return ""
}
