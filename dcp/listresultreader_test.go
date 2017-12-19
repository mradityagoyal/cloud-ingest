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

	_, _, err := reader.ReadLines(context.Background(), "bucket", "object", 0, 5)

	if err == nil {
		t.Errorf("Expected error '%v', but got <nil>", storage.ErrObjectNotExist)
	}
}

func TestReadListResultSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := gcloud.NewMockGCS(mockCtrl)

	src := helpers.NewStringReadCloser("junkid\nline1\nline2\nline3\nline4")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(0)
	lines, endingOffset, err := reader.ReadLines(context.Background(), "bucket", "object", startingOffset, 3)
	if err != nil {
		t.Errorf("Expected no error, but got '%v'", err)
	}

	expectedLines := []string{"line1", "line2", "line3"}
	if len(expectedLines) != len(lines) {
		t.Errorf("Wrong number of lines returned (actual %v vs expected %v)", len(lines), len(expectedLines))
	} else {
		for i := range lines {
			if lines[i] != expectedLines[i] {
				t.Errorf("Line %v doesn't match expectation (actual %v vs expected %v)", i, lines[i], expectedLines[i])
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

	src := helpers.NewStringReadCloser("line2\nline3\nline4\nline5\bline6")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	// Note that since the StringReadCloser is mocked here, the "line2\nline3..." is what is
	// returned as if the startingOffset was 12.
	startingOffset := int64(12)
	lines, endingOffset, err := reader.ReadLines(context.Background(), "bucket", "object", startingOffset, 3)
	if err != nil {
		t.Errorf("Expected no error, but got '%v'", err)
	}

	expectedLines := []string{"line2", "line3", "line4"}
	if len(expectedLines) != len(lines) {
		t.Errorf("Wrong number of lines returned (actual %v vs expected %v)", len(lines), len(expectedLines))
	} else {
		for i := range lines {
			if lines[i] != expectedLines[i] {
				t.Errorf("Line %v doesn't match expectation (actual %v vs expected %v)", i, lines[i], expectedLines[i])
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

	src := helpers.NewStringReadCloser("junkid\nline with spaces\nline2")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(0)
	lines, endingOffset, err := reader.ReadLines(context.Background(), "bucket", "object", startingOffset, 3)
	if err != io.EOF {
		t.Errorf("Expected io.EOF error, but got '%v'", err)
	}

	expectedLines := []string{"line with spaces", "line2"}
	if len(expectedLines) != len(lines) {
		t.Errorf("Wrong number of lines returned (actual %v vs expected %v)", len(lines), len(expectedLines))
	} else {
		for i := range lines {
			if lines[i] != expectedLines[i] {
				t.Errorf("Line %v doesn't match expectation (actual %v vs expected %v)", i, lines[i], expectedLines[i])
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

	src := helpers.NewStringReadCloser("some line\nthe last line")
	mockGcs.EXPECT().
		NewRangeReader(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	startingOffset := int64(123)
	lines, endingOffset, err := reader.ReadLines(context.Background(), "bucket", "object", startingOffset, 2)
	if err != io.EOF {
		t.Errorf("Expected io.EOF error, but got '%v'", err)
	}

	expectedLines := []string{"some line", "the last line"}
	if len(expectedLines) != len(lines) {
		t.Errorf("Wrong number of lines returned (actual %v vs expected %v)", len(lines), len(expectedLines))
	} else {
		for i := range lines {
			if lines[i] != expectedLines[i] {
				t.Errorf("Line %v doesn't match expectation (actual %v vs expected %v)", i, lines[i], expectedLines[i])
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
