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

package agent

import (
	"testing"
	"testing/iotest"

	"github.com/golang/protobuf/proto"

	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
)

type bufferReader struct {
	buf []byte
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *bufferReader) Read(result []byte) (int, error) {
	var i int
	for i = 0; i < min(len(r.buf), len(result)); i++ {
		result[i] = r.buf[i]
	}
	r.buf = r.buf[i:]
	return i, nil
}

type bufferWriter struct {
	buf []byte
}

func (w *bufferWriter) Write(input []byte) (int, error) {
	for _, b := range input {
		w.buf = append(w.buf, b)
	}
	return len(input), nil
}

func TestWriteAndReadSingleProtobuf(t *testing.T) {
	jobRunVersion := "1.0.0"
	header := &listpb.ListFileHeader{JobRunVersion: jobRunVersion}
	w := &bufferWriter{buf: make([]byte, 0)}
	err := writeProtobuf(w, header)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	r := &bufferReader{buf: w.buf}
	readHeader := &listpb.ListFileHeader{}
	err = parseProtobuf(r, readHeader)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if !proto.Equal(readHeader, header) {
		t.Errorf("Expected %v, actual %v", header, readHeader)
	}
}

func TestWriteAndReadManyMixedProtobufs(t *testing.T) {
	protobufs := make([]proto.Message, 0)
	results := make([]proto.Message, 0)
	header1 := &listpb.ListFileHeader{JobRunVersion: "1.0.0"}
	w := &bufferWriter{buf: make([]byte, 0)}
	err := writeProtobuf(w, header1)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, header1)
	results = append(results, &listpb.ListFileHeader{})

	fileInfo1 := &listpb.FileInfo{Path: "Path/to/file/1", LastModifiedTime: 123456, Size: 5}
	err = writeProtobuf(w, fileInfo1)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, fileInfo1)
	results = append(results, &listpb.FileInfo{})

	fileInfo2 := &listpb.FileInfo{Path: "Path/to/file/2", LastModifiedTime: 12345, Size: 25}
	err = writeProtobuf(w, fileInfo2)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, fileInfo2)
	results = append(results, &listpb.FileInfo{})

	header2 := &listpb.ListFileHeader{JobRunVersion: "1.0.1"}
	err = writeProtobuf(w, header2)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, header2)
	results = append(results, &listpb.ListFileHeader{})

	directory := &listpb.DirectoryInfo{Path: "directoryName"}
	err = writeProtobuf(w, directory)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	protobufs = append(protobufs, directory)
	results = append(results, &listpb.DirectoryInfo{})

	r := &bufferReader{buf: w.buf}
	for i, protobuf := range protobufs {
		err = parseProtobuf(r, results[i])
		if err != nil {
			t.Fatalf("Got error %v", err)
		}
		if !proto.Equal(protobuf, results[i]) {
			t.Fatalf("Expected %v, instead got %v", protobuf, results[i])
		}
	}
}

func TestWriteAndReadProtobufsIncompleteRead(t *testing.T) {
	jobRunVersion := "1.0.0"
	header := &listpb.ListFileHeader{JobRunVersion: jobRunVersion}
	w := &bufferWriter{buf: make([]byte, 0)}
	err := writeProtobuf(w, header)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	r := iotest.HalfReader(&bufferReader{buf: w.buf})
	readHeader := &listpb.ListFileHeader{}
	err = parseProtobuf(r, readHeader)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if !proto.Equal(readHeader, header) {
		t.Errorf("Expected %v, actual %v", header, readHeader)
	}
}
