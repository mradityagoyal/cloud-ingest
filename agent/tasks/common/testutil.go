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

package common

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"testing"

	"cloud.google.com/go/storage"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func CheckFailureWithType(taskRelRsrcName string, failureType taskpb.FailureType, taskRespMsg *taskpb.TaskRespMsg, t *testing.T) {
	if taskRespMsg.TaskRelRsrcName != taskRelRsrcName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRelRsrcName, taskRespMsg.TaskRelRsrcName)
	}
	if taskRespMsg.Status != "FAILURE" {
		t.Errorf("want task fail, found: %s", taskRespMsg.Status)
	}
	if taskRespMsg.FailureType != failureType {
		t.Errorf("want task to fail with %s type, got: %s",
			taskpb.FailureType_name[int32(failureType)],
			taskpb.FailureType_name[int32(taskRespMsg.FailureType)])
	}
}

func CheckSuccessMsg(taskRelRsrcName string, taskRespMsg *taskpb.TaskRespMsg, t *testing.T) {
	if taskRespMsg.TaskRelRsrcName != taskRelRsrcName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRelRsrcName, taskRespMsg.TaskRelRsrcName)
	}
	if taskRespMsg.Status != "SUCCESS" {
		t.Errorf("want message success, got: %s", taskRespMsg.Status)
	}
}

type stringReadCloser struct {
	reader io.Reader
	Closed bool
}

func (src *stringReadCloser) Read(p []byte) (int, error) {
	return src.reader.Read(p)
}

func (src *stringReadCloser) Close() error {
	src.Closed = true
	return nil
}

func NewStringReadCloser(s string) *stringReadCloser {
	return &stringReadCloser{strings.NewReader(s), false}
}

// StringWriteCloser implements WriteCloser interface for faking storage.Writer.
type StringWriteCloser struct {
	buffer bytes.Buffer
	closed bool

	// attrs fakes the storage object attributes generated after write completion.
	attrs *storage.ObjectAttrs
}

func NewStringWriteCloser(attrs *storage.ObjectAttrs) *StringWriteCloser {
	return &StringWriteCloser{attrs: attrs}
}

func (m *StringWriteCloser) Write(p []byte) (int, error) {
	return m.buffer.Write(p)
}

func (m *StringWriteCloser) Close() error {
	m.closed = true
	return nil
}

func (m *StringWriteCloser) CloseWithError(err error) error {
	return nil
}

func (m *StringWriteCloser) Attrs() *storage.ObjectAttrs {
	if m.closed {
		return m.attrs
	}
	return nil
}

func (m *StringWriteCloser) WrittenString() string {
	if m.closed {
		return m.buffer.String()
	}
	return ""
}

func (m *StringWriteCloser) NumberLines() int64 {
	if m.closed {
		return CountLines(m.buffer.String())
	}
	return int64(0)
}

func CountLines(s string) int64 {
	return int64(len(strings.Split(strings.Trim(s, "\n"), "\n")))
}

// CreateTmpFile creates a new temporary file in the directory dir with a name
// beginning with prefix, and a content string. If dir is the empty string,
// CreateTmpFile uses the default directory for temporary files (see os.TempDir).
// This method will return the path of the created file. It will panic in case
// of failure creating or writing to the file.
func CreateTmpFile(dir, filePrefix, content string) string {
	tmpfile, err := ioutil.TempFile(dir, filePrefix)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	return tmpfile.Name()
}

// CreateTmpDir creates a new temporary directory in the directory dir with a
// name beginning with prefix and returns the path of the new directory. If dir
// is the empty string, CreateTmpDir uses the default directory for temporary
// files (see os.TempDir).
func CreateTmpDir(dir, prefix string) string {
	tmpDir, err := ioutil.TempDir(dir, prefix)
	if err != nil {
		log.Fatal(err)
	}
	return tmpDir
}
