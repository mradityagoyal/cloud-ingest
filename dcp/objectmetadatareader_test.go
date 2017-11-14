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
	"cloud.google.com/go/storage"
	"errors"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestGCSError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGcs := NewMockGCS(mockCtrl)
	mockGcs.EXPECT().GetAttrs(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))

	reader := NewGCSObjectMetadataReader(mockGcs)

	_, err := reader.GetMetadata("bucket", "object")
	if err == nil {
		t.Error("Error should not be nil")
	}
}

func TestMtimeMissing(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	attr := storage.ObjectAttrs{
		Size:       123,
		Generation: 234,
		Metadata:   map[string]string{},
	}

	mockGcs := NewMockGCS(mockCtrl)
	mockGcs.EXPECT().GetAttrs(gomock.Any(), gomock.Any()).Return(&attr, nil)

	reader := NewGCSObjectMetadataReader(mockGcs)

	result, err := reader.GetMetadata("bucket", "object")

	expected := &ObjectMetadata{Size: 123, Mtime: 0, GenerationNumber: 234}

	if err != nil {
		t.Errorf("Error should be nil, but was %v", err)
	} else if *expected != *result {
		t.Errorf("Wrong result: wanted %v, but got %v", expected, result)
	}
}

func TestMtimeMisformatted(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	attr := storage.ObjectAttrs{
		Size:       123,
		Generation: 234,
		Metadata:   map[string]string{MTIME_ATTR_NAME: "totally not a number"},
	}

	mockGcs := NewMockGCS(mockCtrl)
	mockGcs.EXPECT().GetAttrs(gomock.Any(), gomock.Any()).Return(&attr, nil)

	reader := NewGCSObjectMetadataReader(mockGcs)

	_, err := reader.GetMetadata("bucket", "object")
	if err == nil {
		t.Error("Error should not be nil")
	}

}

func TestSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	attr := storage.ObjectAttrs{
		Size:       123,
		Generation: 234,
		Metadata:   map[string]string{MTIME_ATTR_NAME: "345"},
	}

	expected := &ObjectMetadata{Size: 123, Mtime: 345, GenerationNumber: 234}

	mockGcs := NewMockGCS(mockCtrl)
	mockGcs.EXPECT().GetAttrs(gomock.Any(), gomock.Any()).Return(&attr, nil)

	reader := NewGCSObjectMetadataReader(mockGcs)

	result, err := reader.GetMetadata("bucket", "object")

	if err != nil {
		t.Errorf("Error should be nil, but was %v", err)
	} else if *expected != *result {
		t.Errorf("Wrong result: wanted %v, but got %v", expected, result)
	}
}
