package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/release/changelog"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"google.golang.org/api/option"
)

const (
	dev  = "dev"
	prod = "prod"
)

var buildType string

func init() {
	flag.StringVar(&buildType, "buildType", "", fmt.Sprintf("The type of build. Should be %q or %q. (Required)", dev, prod))
	flag.Parse()
}

func main() {
	if buildType == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Generate json bytes for the current changelog file
	out, err := changelog.ParseChangelogFile(path.Join(changelog.GoPath(), changelog.RepoPath, "CHANGELOG.md"))
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

	if buildType != dev {
		// Fetch the version of the most recently released agent
		ctx := context.Background()
		client, err := storage.NewClient(ctx, option.WithoutAuthentication())
		if err != nil {
			log.Fatalf("Unable to create GCS client. Error: %v", err)
		}
		prodVersion, err := changelog.FetchVersion(ctx, stiface.AdaptClient(client), changelog.ProdBucketName, changelog.ProdObjectName)
		if err != nil {
			log.Fatalf("Unable to fetch prod version. Error: %v", err)
		}

		err = changelog.ValidateRelease(versions, prodVersion)
		if err != nil {
			log.Fatalf("Release validation failed. Error: %v", err)
		}
	}

	// Handle the unreleased cases
	for versions[0].Version == "" {
		versions = versions[1:]
	}

	fmt.Println("Changelog validation passed!")
	fmt.Printf("Newest version is:\nv%s\n" ,strings.Trim(versions[0].Version, "{}"))
}