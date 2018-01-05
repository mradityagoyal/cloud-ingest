/*
Copyright 2017 Google Inc. All Rights Reserved.
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

package dcp

import (
	"context"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
)

func TestReadListResultError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, storage.ErrObjectNotExist)

	reader := NewGCSListingResultReader(mockGcs)

	_, _, err := reader.ReadEntries(context.Background(), "bucket", "object", 0, 5)

	if err == nil {
		t.Errorf("Expected error '%v', but got <nil>", storage.ErrObjectNotExist)
	}
}

func TestReadListResultSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)

	src := helpers.NewStringReadCloser("junkid\nd,line1\nf,line2\nf,line3\nf,line4")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(0)
	listFileEntries, endingOffset, err := reader.ReadEntries(context.Background(), "bucket", "object", startingOffset, 3)
	if err != nil {
		t.Errorf("Expected no error, but got '%v'", err)
	}

	expectedEntries := []ListFileEntry{
		ListFileEntry{true, "line1"},
		ListFileEntry{false, "line2"},
		ListFileEntry{false, "line3"},
	}
	if len(expectedEntries) != len(listFileEntries) {
		t.Errorf("Wrong number of listFileEntries returned (actual %v vs expected %v)", len(listFileEntries), len(expectedEntries))
	} else {
		for i := range listFileEntries {
			if listFileEntries[i] != expectedEntries[i] {
				t.Errorf("Entry %v doesn't match expectation (actual %v vs expected %v)", i, listFileEntries[i], expectedEntries[i])
			}
		}
	}

	if endingOffset <= startingOffset {
		t.Errorf("Expected endingOffset > startingOffset (endingOffset %v vs startingOffset %v)", endingOffset, startingOffset)
	}

	if !src.Closed {
		t.Error("Did not close the reader.")
	}
}

func TestReadListResultNonzeroOffset(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)

	src := helpers.NewStringReadCloser("d,line2\nf,line3\nf,line4\nf,line5\bf,line6")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	// Note that since the StringReadCloser is mocked here, the "line2\nline3..." is what is
	// returned as if the startingOffset was 12.
	startingOffset := int64(12)
	listFileEntries, endingOffset, err := reader.ReadEntries(context.Background(), "bucket", "object", startingOffset, 3)
	if err != nil {
		t.Errorf("Expected no error, but got '%v'", err)
	}

	expectedEntries := []ListFileEntry{
		ListFileEntry{true, "line2"},
		ListFileEntry{false, "line3"},
		ListFileEntry{false, "line4"},
	}
	if len(expectedEntries) != len(listFileEntries) {
		t.Errorf("Wrong number of listFileEntries returned (actual %v vs expected %v)", len(listFileEntries), len(expectedEntries))
	} else {
		for i := range listFileEntries {
			if listFileEntries[i] != expectedEntries[i] {
				t.Errorf("Entry %v doesn't match expectation (actual %v vs expected %v)", i, listFileEntries[i], expectedEntries[i])
			}
		}
	}

	if endingOffset <= startingOffset {
		t.Errorf("Expected endingOffset > startingOffset (endingOffset %v vs startingOffset %v)", endingOffset, startingOffset)
	}

	if !src.Closed {
		t.Error("Did not close the reader.")
	}
}

func TestReadListResultEOF(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)

	src := helpers.NewStringReadCloser("junkid\nd,line with spaces\nd,line2")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(0)
	listFileEntries, endingOffset, err := reader.ReadEntries(context.Background(), "bucket", "object", startingOffset, 3)
	if err != io.EOF {
		t.Errorf("Expected io.EOF error, but got '%v'", err)
	}

	expectedEntries := []ListFileEntry{
		ListFileEntry{true, "line with spaces"},
		ListFileEntry{true, "line2"},
	}
	if len(expectedEntries) != len(listFileEntries) {
		t.Errorf("Wrong number of listFileEntries returned (actual %v vs expected %v)", len(listFileEntries), len(expectedEntries))
	} else {
		for i := range listFileEntries {
			if listFileEntries[i] != expectedEntries[i] {
				t.Errorf("Entry %v doesn't match expectation (actual %v vs expected %v)", i, listFileEntries[i], expectedEntries[i])
			}
		}
	}

	if endingOffset <= startingOffset {
		t.Errorf("Expected endingOffset > startingOffset (endingOffset %v vs startingOffset %v)", endingOffset, startingOffset)
	}

	if !src.Closed {
		t.Error("Did not close the reader.")
	}
}

func TestReadListResultMaxLinesEOF(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)

	src := helpers.NewStringReadCloser("f,some line\nf,the last line")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(123)
	listFileEntries, endingOffset, err := reader.ReadEntries(context.Background(), "bucket", "object", startingOffset, 2)
	if err != io.EOF {
		t.Errorf("Expected io.EOF error, but got '%v'", err)
	}

	expectedEntries := []ListFileEntry{
		ListFileEntry{false, "some line"},
		ListFileEntry{false, "the last line"},
	}
	if len(expectedEntries) != len(listFileEntries) {
		t.Errorf("Wrong number of listFileEntries returned (actual %v vs expected %v)", len(listFileEntries), len(expectedEntries))
	} else {
		for i := range listFileEntries {
			if listFileEntries[i] != expectedEntries[i] {
				t.Errorf("Line %v doesn't match expectation (actual %v vs expected %v)", i, listFileEntries[i], expectedEntries[i])
			}
		}
	}

	if endingOffset <= startingOffset {
		t.Errorf("Expected endingOffset > startingOffset (endingOffset %v vs startingOffset %v)", endingOffset, startingOffset)
	}

	if !src.Closed {
		t.Error("Did not close the reader.")
	}
}

func TestListFileEntryParseAndStringSuccess(t *testing.T) {
	var tests = []struct {
		entry ListFileEntry
		line  string
	}{
		{ListFileEntry{true, "some path"}, "d,some path"},
		{ListFileEntry{true, "a/b/c/d"}, "d,a/b/c/d"},
		{ListFileEntry{true, "/a/b/c/d"}, "d,/a/b/c/d"},
		{ListFileEntry{true, "//a/b/c/d"}, "d,//a/b/c/d"},
		{ListFileEntry{true, "a\\b\\c\\d"}, "d,a\\b\\c\\d"},
		{ListFileEntry{true, "c:\\a\\b\\c\\d"}, "d,c:\\a\\b\\c\\d"},
		{ListFileEntry{true, "a,b"}, "d,a,b"},
		{ListFileEntry{false, "some path"}, "f,some path"},
		{ListFileEntry{false, "a/b/c/d"}, "f,a/b/c/d"},
		{ListFileEntry{false, "/a/b/c/d"}, "f,/a/b/c/d"},
		{ListFileEntry{false, "//a/b/c/d"}, "f,//a/b/c/d"},
		{ListFileEntry{false, "a\\b\\c\\d"}, "f,a\\b\\c\\d"},
		{ListFileEntry{false, "c:\\a\\b\\c\\d"}, "f,c:\\a\\b\\c\\d"},
		{ListFileEntry{false, "a,b"}, "f,a,b"},
	}
	for _, tc := range tests {
		parsedEntry, err := ParseListFileLine(tc.line)
		if err != nil {
			t.Errorf("Error parsing line %v, err: %v", tc.line, err)
		}
		if *parsedEntry != tc.entry {
			t.Errorf("Expected parsed %v, actual: %v", tc.entry, *parsedEntry)
		}
		if s := tc.entry.String(); s != tc.line {
			t.Errorf("Expected entry string %v, actual: %v", tc.line, s)
		}
	}
}

func TestListFileEntryParseFailure(t *testing.T) {
	// Parse fails without the correct number of fields.
	expectedErr := "expected 2 fields"
	if _, err := ParseListFileLine("some path with no delimiter"); err == nil {
		t.Errorf("error is nil, expected error: %v...", expectedErr)
	} else if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %s, found: %s.", expectedErr, err.Error())
	}

	// Parse fails with a bogus type field.
	expectedErr = "expected 'd' or 'f'"
	if _, err := ParseListFileLine("b,bogus type field"); err == nil {
		t.Errorf("error is nil, expected error: %v...", expectedErr)
	} else if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %s, found: %s.", expectedErr, err.Error())
	}
}
