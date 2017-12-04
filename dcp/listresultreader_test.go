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
	"reflect"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
)

func TestReadListResultError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := NewMockGCS(mockCtrl)
	mockGcs.EXPECT().NewReader(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, storage.ErrObjectNotExist)

	reader := NewGCSListingResultReader(mockGcs)

	_, err := reader.ReadListResult(context.Background(), "bucket", "object")

	if err == nil {
		t.Errorf("Expected error '%v', but got <nil>", storage.ErrObjectNotExist)
	}
}

func TestReadListResultSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := NewMockGCS(mockCtrl)

	src := NewStringReadCloser("line1\nline2\n")
	mockGcs.EXPECT().
		NewReader(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(src, nil)

	reader := NewGCSListingResultReader(mockGcs)

	result, err := reader.ReadListResult(context.Background(), "bucket", "object")

	if err != nil {
		t.Errorf("Expected no error, but got '%v'", err)
	}

	lines := make([]string, 0)
	for line := range result {
		lines = append(lines, line)
	}

	expected := []string{"line1", "line2"}
	if !reflect.DeepEqual(expected, lines) {
		t.Errorf("Expected %v, but got %v", expected, lines)
	}

	if !src.closed {
		t.Error("Did not close the reader.")
	}
}
