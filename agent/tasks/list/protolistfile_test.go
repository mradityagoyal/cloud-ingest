/*
Copyright 2018 Google Inc. All Rights Reserved.
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

package list

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"

	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
)

func TestWriteAndReadSingleProtobuf(t *testing.T) {
	path := "an/example/path"
	dirInfo := &listpb.DirectoryInfo{Path: path}
	var buf bytes.Buffer
	err := writeProtobuf(&buf, dirInfo)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	readDirInfo := &listpb.DirectoryInfo{}
	err = parseProtobuf(&buf, readDirInfo)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if !proto.Equal(readDirInfo, dirInfo) {
		t.Errorf("Expected %v, actual %v", dirInfo, readDirInfo)
	}
}

func TestWriteAndReadManyMixedProtobufs(t *testing.T) {
	protobufs := make([]proto.Message, 0)
	results := make([]proto.Message, 0)
	var buf bytes.Buffer

	fileInfo1 := &listpb.FileInfo{Path: "Path/to/file/1", LastModifiedTime: 123456, Size: 5}
	err := writeProtobuf(&buf, fileInfo1)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, fileInfo1)
	results = append(results, &listpb.FileInfo{})

	fileInfo2 := &listpb.FileInfo{Path: "Path/to/file/2", LastModifiedTime: 12345, Size: 25}
	err = writeProtobuf(&buf, fileInfo2)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, fileInfo2)
	results = append(results, &listpb.FileInfo{})

	directory := &listpb.DirectoryInfo{Path: "directoryName"}
	err = writeProtobuf(&buf, directory)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, directory)
	results = append(results, &listpb.DirectoryInfo{})

	for i, protobuf := range protobufs {
		err = parseProtobuf(&buf, results[i])
		if err != nil {
			t.Fatalf("Got error %v", err)
		}
		if !proto.Equal(protobuf, results[i]) {
			t.Fatalf("Expected %v, instead got %v", protobuf, results[i])
		}
	}
}

func TestWriteAndReadProtobufsIncompleteRead(t *testing.T) {
	path := "an/example/path"
	dirInfo := &listpb.DirectoryInfo{Path: path}
	var buf bytes.Buffer
	err := writeProtobuf(&buf, dirInfo)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	readDirInfo := &listpb.DirectoryInfo{Path: path}
	err = parseProtobuf(&buf, readDirInfo)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if !proto.Equal(readDirInfo, dirInfo) {
		t.Errorf("Expected %v, actual %v", dirInfo, readDirInfo)
	}
}
