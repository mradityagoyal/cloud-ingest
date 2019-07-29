package common

import (
	"flag"
	"fmt"
	"os"

	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

var (
	AgentIDPrefix = flag.String("agent-id-prefix", "", "A a prefix to include in the agent ID.")
	ContainerID   = flag.String("container-id", "", "The container ID. This flag is only populated when the agent binary is running inside a container.")
	hostname      = flag.String("hostname", "hostnameunknown", "Hostname of the container host machine. This flag is only required when the agent binary is running inside a container.")
)

// Hostname returns the hostname string.
func Hostname() string {
	// Use the hostname flag value if the agent is running inside a container.
	if *ContainerID != "" {
		return *hostname
	}
	hn, err := os.Hostname()
	if err != nil {
		hn = "hostnameunknown"
	}
	return hn
}

// ProcessID returns the PID as a string.
func ProcessID() string {
	// Only set PID when agent is not running inside a container.
	if *ContainerID != "" {
		return ""
	}
	return fmt.Sprintf("%v", os.Getpid())
}

// AgentID returns the ID of this agent.
func AgentID() *pulsepb.AgentId {
	return &pulsepb.AgentId{
		HostName:    Hostname(),
		ProcessId:   ProcessID(),
		Prefix:      *AgentIDPrefix,
		ContainerId: *ContainerID,
	}
}
