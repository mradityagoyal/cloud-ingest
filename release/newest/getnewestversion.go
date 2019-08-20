package newest

import (
	"fmt"
	"log"
	"path"
	"strings"


	"github.com/GoogleCloudPlatform/cloud-ingest/release/changelog"
)

func main() {
  // Generate json bytes for the current changelog file
  out, err := changelog.ParseChangelogFile(
    path.Join(changelog.GoPath(), changelog.RepoPath, "CHANGELOG.md"))
  if err != nil {
    log.Fatalf("Failed to parse changelog and create json representation. Error: %v", err)
  }

  versions, err := changelog.ParseChangelogVersions(out)
  if err != nil {
    log.Fatalf("Failed to parse changelog versions. Error: %v", err)
  }

  err = changelog.ValidateChangelogVersions(versions)
  if err != nil {
    log.Fatalf("Changelog validation failed. Error: %v", err)
  }

  // Handle the unreleased case
  if versions[0].Version == "" {
      versions = versions[1:]
  }

  // Newest version in changelog in standard agent version format
  fmt.Printf("v%s\n", strings.Trim(versions[0].Version, "{}"))
}