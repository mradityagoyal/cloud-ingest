package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/blang/semver"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"google.golang.org/api/option"
)

var repoPath = path.Join("src", "github.com", "GoogleCloudPlatform", "cloud-ingest")

const bucketName = "cloud-ingest-pub"
const objectName = "agent/current/VERSIONINFO.txt"

type Changelog struct {
	Versions []CLVersion `json:"versions"`
}

type CLVersion struct {
	Version string `json:"version,omitempty"`
}

func main() {
	// Generate json bytes for the current changelog file
	out := parseChangelogFile(path.Join(goPath(), repoPath, "CHANGELOG.md"))

	// Fetch the version of the most recently released agent
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		log.Fatalf("Unable to create GCS client. Error: %v", err)
	}
	prodVersion := fetchProdVersion(ctx, stiface.AdaptClient(client))

	versions := parseChangelogVersions(out)

	err = validateChangelogVersions(versions, prodVersion)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Changelog validation passed!")
}

func goPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}

func parseChangelogFile(filePath string) []byte {
	cmd := exec.Command(path.Join(goPath(), repoPath, "node_modules", "changelog-parser", "bin", "cli.js"), filePath)
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to parse changelog and create json representation. Error: %v", err)
	}
	return out
}

func fetchProdVersion(ctx context.Context, client stiface.Client) semver.Version {
	bkt := client.Bucket(bucketName)
	obj := bkt.Object(objectName)
	r, err := obj.NewReader(ctx)
	if err != nil {
		log.Fatalf("Failed to read current agent's version. Error: %v", err)
	}
	defer r.Close()
	reader := bufio.NewReader(r)
	versionLine, isPrefix, err := reader.ReadLine()
	if err != nil {
		log.Fatalf("Failed to read version line. Error: %v", err)
	}
	if isPrefix {
		log.Fatalf("Failed to read version line; the line was too long.")
	}
	parts := strings.Split(string(versionLine), ":")
	if len(parts) != 2 {
		log.Fatalf("VERSIONINFO.txt was not formatted correctly.")
	}
	version, err := semver.ParseTolerant(parts[1])
	if err != nil {
		log.Fatalf("Failed to parse version %q with error %v", parts[1], err)
	}
	return version
}

func parseChangelogVersions(jsonBytes []byte) []CLVersion {
	changelog := &Changelog{}
	err := json.Unmarshal(jsonBytes, changelog)
	if err != nil {
		log.Fatalf("Failed to unmarshal json object. Error: %v", err)
	}
	return changelog.Versions
}

func validateChangelogVersions(clVersions []CLVersion, currentVersion semver.Version) error {
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

	// Check that the most recent version in the changelog is greater than the currently released version.
	lastVersion := versions[0]
	if lastVersion.Major > 0 && !lastVersion.GT(currentVersion) {
		return fmt.Errorf("most recent version in the changelog, %q, is not greater than the currently released version, %q", lastVersion.String(), currentVersion.String())
	}

	// Make sure the versions in the changelog are sorted in increasing order
	for _, v := range versions[1:] {
		if !lastVersion.GT(v) {
			return fmt.Errorf("version %q comes after version %q in the changelog. This is a violation of semver", lastVersion.String(), v.String())
		}
		lastVersion = v
	}
	return nil
}
