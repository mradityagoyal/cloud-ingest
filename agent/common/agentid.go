package common

import (
	"flag"
	"fmt"
	"os"

	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

var (
	AgentIDPrefix = flag.String("agent-id-prefix", "", "A a prefix to include in the agent ID.")
)

// Hostname returns the hostname string.
func Hostname() string {
	hn, err := os.Hostname()
	if err != nil {
		hn = "hostnameunknown"
	}
	return hn
}

// AgentID returns the ID of this agent.
func AgentID() *pulsepb.AgentId {
	return &pulsepb.AgentId{
		HostName:  Hostname(),
		ProcessId: fmt.Sprintf("%v", os.Getpid()),
		Prefix:    *AgentIDPrefix,
	}
}
