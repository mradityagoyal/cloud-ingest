// Package changelog contains functions to parse and validate the OPT changelog
package changelog

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/blang/semver"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
)

var RepoPath = path.Join("src", "github.com", "GoogleCloudPlatform", "cloud-ingest")

const ProdBucketName = "cloud-ingest-pub"
const ProdObjectName = "agent/current/VERSIONINFO.txt"

type Changelog struct {
	Versions []CLVersion `json:"versions"`
}

type CLVersion struct {
	Version string `json:"version,omitempty"`
}

func GoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}

// ParseChangelogFile parses the changelog in the given file and returns the JSON output as a
// slice of bytes. If the function is unable to parse the given changelog, an error is returned.
func ParseChangelogFile(filePath string) ([]byte, error) {
	cmd := exec.Command(path.Join(GoPath(), RepoPath, "node_modules", "changelog-parser", "bin", "cli.js"), filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FetchVersion fetches the version stored in the specified VERSIONINFO.txt GCS object.
// Expects the first line of the file to be formatted as follows:
//     Version: <version number, e.g. v0.5.2>
// FetchVersion returns an error if it is unable to fetch the version.
func FetchVersion(ctx context.Context, client stiface.Client, bucketName, objectName string) (semver.Version, error) {
	bkt := client.Bucket(bucketName)
	obj := bkt.Object(objectName)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return semver.Version{}, fmt.Errorf("error reading current agent's version: %v", err)
	}
	defer r.Close()
	reader := bufio.NewReader(r)
	versionLine, isPrefix, err := reader.ReadLine()
	if err != nil {
		return semver.Version{}, fmt.Errorf("error reading VERSIONINFO.txt version line: %v", err)
	}
	if isPrefix {
		return semver.Version{}, errors.New("VERSIONINFO.txt version line was too long")
	}
	parts := strings.Split(string(versionLine), ":")
	if len(parts) != 2 {
		return semver.Version{}, errors.New("VERSIONINFO.txt was not formatted correctly.")
	}
	version, err := semver.ParseTolerant(parts[1])
	if err != nil {
		return semver.Version{}, fmt.Errorf("Failed to parse version %q with error %v", parts[1], err)
	}
	return version, nil
}

// ParseChangelogVersions parses versions from the given json representation of a changelog.
// An error is returned if it is unable to parse the versions.
func ParseChangelogVersions(jsonBytes []byte) ([]CLVersion, error) {
	changelog := &Changelog{}
	err := json.Unmarshal(jsonBytes, changelog)
	if err != nil {
		return nil, err
	}
	return changelog.Versions, nil
}

// ValidateChangelogVersions validates the given changelog versions, ensuring that the given
// versions are increasing and following the semver spec. Returns an error if the changelog is not
// valid.
func ValidateChangelogVersions(clVersions []CLVersion) error {
	var versions []semver.Version
	if len(clVersions) == 0 {
		return errors.New("changelog is empty")
	}

	// Skip Unreleased section if it exists
	if clVersions[0].Version == "" {
		clVersions = clVersions[1:]
	}

	for _, v := range clVersions {
		version, err := semver.ParseTolerant(v.Version)
		if err != nil {
			return fmt.Errorf("failed to parse version %q with error %v", v.Version, err)
		}
		versions = append(versions, version)
	}

	if len(versions) == 0 {
		return errors.New("changelog contains no versions")
	}

	// Make sure the versions in the changelog are sorted in increasing order
	lastVersion := versions[0]
	for _, v := range versions[1:] {
		if !lastVersion.GT(v) {
			return fmt.Errorf("version %q comes after version %q in the changelog. This is a violation of semver", lastVersion.String(), v.String())
		}
		lastVersion = v
	}
	return nil
}

// ValidateRelease ensures that the latest version in the changelog is not less than or equal to
// the currently released version. If the release version is not valid, or if an error occurs, an
// error is returned.
func ValidateRelease(clVersions []CLVersion, currentVersion semver.Version) error {
	if len(clVersions) == 0 {
		return errors.New("changelog is empty")
	}

	// Skip Unreleased section if it exists
	if clVersions[0].Version == "" {
		clVersions = clVersions[1:]
	}

	if len(clVersions) == 0 {
		return errors.New("changelog doesn't contain a version")
	}

	// Check that the most recent version in the changelog is greater than the currently released version.
	lastVersion, err := semver.ParseTolerant(clVersions[0].Version)
	if err != nil {
		return err
	}
	if lastVersion.Major > 0 && !lastVersion.GT(currentVersion) {
		return fmt.Errorf("most recent version in the changelog, %q, is not greater than the currently released version, %q", lastVersion.String(), currentVersion.String())
	}
	return nil
}
