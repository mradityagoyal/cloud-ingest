package main

import (
	"bytes"
	"context"
	"path"
	"testing"

	"github.com/blang/semver"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
)

const sampleVersionInfoFile = `Version: v0.5.2
Commit: 8531c7809b318cec4d5d0f5e60245c98a11bace3
Build Date: 2018-11-19T23:01:30UTC`

const sampleChanglogFileName = "changelogtestsample.md"

func TestFetchProdVersion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := stiface.NewMockClient(mockCtrl)
	mockBucket := stiface.NewMockBucketHandle(mockCtrl)
	mockObj := stiface.NewMockObjectHandle(mockCtrl)

	r := bytes.NewBuffer([]byte(sampleVersionInfoFile))
	mockReader := stiface.NewMockReader(mockCtrl)
	mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
		return r.Read(p)
	})
	mockReader.EXPECT().Close()
	ctx := context.Background()
	mockObj.EXPECT().NewReader(ctx).Return(mockReader, nil)

	mockBucket.EXPECT().Object(objectName).Return(mockObj)
	mockClient.EXPECT().Bucket(bucketName).Return(mockBucket)

	want := semver.Version{Major: 0, Minor: 5, Patch: 2}
	got := fetchProdVersion(ctx, mockClient)
	if !cmp.Equal(got, want) {
		t.Errorf("fetchProdVersion() = %q, want %q", got.String(), want.String())
	}
}

func TestParseChangelogVersions(t *testing.T) {
	out := parseChangelogFile(path.Join(goPath(), repoPath, "release", sampleChanglogFileName))
	got := parseChangelogVersions(out)
	want := []CLVersion{
		{},
		{Version: "1.0.0"},
	}
	if !cmp.Equal(got, want) {
		t.Errorf("parseChangelogVersions() = %v, want %v", got, want)
	}
}

func TestValidateChangelogVersions(t *testing.T) {
	testCases := []struct {
		description    string
		versions       []CLVersion
		currentVersion semver.Version
		wantError      bool
	}{
		{
			description: "Valid changelog",
			versions: []CLVersion{
				{Version: "1.0.0"},
			},
			currentVersion: semver.MustParse("0.0.0"),
			wantError:      false,
		},
		{
			description: "Valid changelog with unreleased section",
			versions: []CLVersion{
				{},
				{Version: "1.0.0"},
			},
			currentVersion: semver.MustParse("0.0.0"),
			wantError:      false,
		},
		{
			description: "build Version equal to prod version",
			versions: []CLVersion{
				{},
				{Version: "1.0.0"},
			},
			currentVersion: semver.MustParse("1.0.0"),
			wantError:      true,
		},
		{
			description: "build version with major version 0 less than prod version",
			versions: []CLVersion{
				{},
				{Version: "0.0.0"},
			},
			currentVersion: semver.MustParse("0.5.2"),
			wantError:      false,
		},
		{
			description: "versions out of order",
			versions: []CLVersion{
				{},
				{Version: "1.0.0"},
				{Version: "1.0.3"},
			},
			currentVersion: semver.MustParse("0.0.0"),
			wantError:      true,
		},
		{
			description: "version format issues",
			versions: []CLVersion{
				{},
				{Version: "5.0.0"},
				{Version: "3.1.0.3"},
			},
			currentVersion: semver.MustParse("3.0.0"),
			wantError:      true,
		},
		{
			description: "missing version",
			versions: []CLVersion{
				{},
				{Version: "5.0.0"},
				{},
			},
			currentVersion: semver.MustParse("3.0.0"),
			wantError:      true,
		},
		{
			description: "empty changelog with unreleased section",
			versions: []CLVersion{
				{},
			},
			currentVersion: semver.MustParse("3.0.0"),
			wantError:      true,
		},
		{
			description:    "empty changelog",
			versions:       []CLVersion{},
			currentVersion: semver.MustParse("3.0.0"),
			wantError:      true,
		},
	}

	for _, test := range testCases {
		err := validateChangelogVersions(test.versions, test.currentVersion)
		if test.wantError && err == nil {
			t.Errorf("validateChangelogVersions(%s) got nil error, want error", test.description)
		}
		if !test.wantError && err != nil {
			t.Errorf("validateChangelogVersions(%s) got error %v", test.description, err)
		}
	}
}
