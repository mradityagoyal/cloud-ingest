package versions

import (
	"github.com/blang/semver"
)

// DefaultJobRunVersion is the job run version to use when the job run version of a task is not specified.
const DefaultJobRunVersion = "0.0.0"

var (
	agentVersion = semver.MustParse(DefaultJobRunVersion)

	supportedJobRuns = []semver.Version{semver.MustParse("0.0.0"), semver.MustParse("1.0.0"), semver.MustParse("2.0.0")}
)

// SetAgentVersion sets the agent version to the given value. If there is an issue parsing
// the version, an error is returned.
func SetAgentVersion(versionStr string) error {
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		return err
	}
	agentVersion = version
	return nil
}

// AgentVersion returns the agent's version.
func AgentVersion() semver.Version {
	return agentVersion
}

// deepCopy makes a deepcopy of the given semver.Version struct.
// A shallow copy works fine for the Major, Minor, and Patch fields. The Build
// and Pre fields use pointers, which is why there is a need for a deepcopy.
func deepCopy(version semver.Version) semver.Version {
	copy := version
	copy.Build = nil
	copy.Build = append(copy.Build, version.Build...)

	copy.Pre = nil
	copy.Pre = append(copy.Pre, version.Pre...)
	return copy
}

// SupportedJobRuns returns a list of the job run versions supported by this agent.
func SupportedJobRuns() []semver.Version {
	var jrvCopy []semver.Version
	for _, jrv := range supportedJobRuns {
		// Do a deepCopy in case we ever decide to use the version build or pre fields.
		jrvCopy = append(jrvCopy, deepCopy(jrv))
	}
	return jrvCopy
}

// VersionFromString returns the version for the given version string. If
// the version is not specified, DefaultJobRunVersion is used.
func VersionFromString(versionStr string) (semver.Version, error) {
	if versionStr == "" {
		versionStr = DefaultJobRunVersion
	}
	return semver.ParseTolerant(versionStr)
}
