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

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"golang.org/x/sync/semaphore"
	raw "google.golang.org/api/storage/v1"
)

const (
	testFileContent   = "Ephemeral test file content for copy_test.go."
	testCRC32C        = 3923584507 // CRC32C of testFileContent.
	testTenByteCRC32C = 1069694901 // CRC32C of the first 10-bytes of testFileContent.
)

func testingTaskReqParams() taskReqParams {
	taskReqParams := make(taskReqParams)
	taskReqParams["src_file"] = "file"
	taskReqParams["dst_bucket"] = "bucket"
	taskReqParams["dst_object"] = "object"
	taskReqParams["expected_generation_num"] = 0
	taskReqParams["file_bytes"] = 0
	taskReqParams["file_mtime"] = 0
	taskReqParams["bytes_copied"] = 0
	taskReqParams["bytes_to_copy"] = 0
	taskReqParams["resumable_upload_id"] = ""
	taskReqParams["crc32c"] = 0
	return taskReqParams
}

func TestNoTaskReqParams(t *testing.T) {
	h := CopyHandler{}
	taskReqParams := taskReqParams{}
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkForInvalidTaskReqParamsArguments("task", msg, t)
}

func TestCopyMissingOneTaskReqParams(t *testing.T) {
	h := &CopyHandler{}
	taskReqParams := testingTaskReqParams()
	testMissingOneTaskReqParams(h, taskReqParams, t)
}

func TestCopyInvalidGenerationNum(t *testing.T) {
	h := CopyHandler{}
	taskReqParams := testingTaskReqParams()
	taskReqParams["expected_generation_num"] = "not a number"
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkForInvalidTaskReqParamsArguments("task", msg, t)
}

func TestSourceNotFound(t *testing.T) {
	h := CopyHandler{}
	taskReqParams := testingTaskReqParams()
	taskReqParams["src_file"] = "file does not exist"
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkFailureWithType("task", proto.TaskFailureType_FILE_NOT_FOUND_FAILURE, msg, t)
}

func TestAcquireBufferMemoryFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{})

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	copyMemoryLimit = 5
	h := CopyHandler{mockGCS, 10, nil, semaphore.NewWeighted(5), nil}
	taskReqParams := testingTaskReqParams()
	taskReqParams["src_file"] = tmpFile
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkFailureWithType("task", proto.TaskFailureType_UNKNOWN, msg, t)
}

func TestCRC32CMismtach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C: 12345, // Incorrect CRC32C.
	})

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	copyMemoryLimit = defaultCopyMemoryLimit
	h := CopyHandler{mockGCS, 5, nil, semaphore.NewWeighted(copyMemoryLimit), nil}
	taskReqParams := testingTaskReqParams()
	taskReqParams["src_file"] = tmpFile
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkFailureWithType("task", proto.TaskFailureType_MD5_MISMATCH_FAILURE, msg, t)
}

func TestCopyEntireFileSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsModTime := time.Now()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C:  uint32(testCRC32C),
		Size:    int64(len(testFileContent)),
		Updated: gcsModTime,
	})

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	copyMemoryLimit = defaultCopyMemoryLimit
	h := CopyHandler{mockGCS, 5, nil, semaphore.NewWeighted(copyMemoryLimit), nil}
	taskReqParams := testingTaskReqParams()
	taskReqParams["src_file"] = tmpFile
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkSuccessMsg("task", msg, t)
	if writer.WrittenString() != testFileContent {
		t.Errorf("written string want \"%s\", got \"%s\"",
			testFileContent, writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLogEntry := dcp.LogEntry{
		"worker_id":         workerID,
		"src_crc32c":        uint32(testCRC32C),
		"dst_crc32c":        uint32(testCRC32C),
		"src_bytes":         int64(len(testFileContent)),
		"dst_bytes":         int64(len(testFileContent)),
		"src_file":          tmpFile,
		"dst_file":          "bucket/object",
		"src_modified_time": srcStats.ModTime(),
		"dst_modified_time": gcsModTime,
	}
	if !reflect.DeepEqual(wantLogEntry, msg.LogEntry) {
		t.Errorf("log entry want: %+v, got: %+v", wantLogEntry, msg.LogEntry)
	}
}

func TestCopyHanderDoResumable(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// This bogus response serves both the prepareResumableCopy and
		// copyResumableChunk requests.
		object := &raw.Object{
			Name:    "dst_o",
			Bucket:  "dst_b",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "modTime",
		}
		body := new(bytes.Buffer)
		_ = json.NewEncoder(body).Encode(object)
		res := &http.Response{
			StatusCode: 200,
			Header:     make(map[string][]string),
			Body:       ioutil.NopCloser(body),
		}
		res.Header.Add("Location", "testResumableUploadId")
		return res, nil
	}

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	taskReqParams := testingTaskReqParams()
	taskReqParams["src_file"] = tmpFile
	taskReqParams["bytes_to_copy"] = 10
	msg := h.Do(context.Background(), "task", taskReqParams)
	checkSuccessMsg("task", msg, t)

	srcStats, _ := os.Stat(tmpFile)
	wantLogEntry := dcp.LogEntry{
		"bytes_copied":      int64(10),
		"dst_file":          "bucket/object",
		"src_bytes":         int64(len(testFileContent)),
		"src_file":          tmpFile,
		"src_modified_time": srcStats.ModTime(),
		"worker_id":         workerID,
	}
	if !reflect.DeepEqual(wantLogEntry, msg.LogEntry) {
		t.Errorf("log entry want: %+v, got: %+v", wantLogEntry, msg.LogEntry)
	}

	wantTaskResParams := taskResParams{
		"bytes_copied":        int64(10),
		"crc32c":              int64(testTenByteCRC32C),
		"file_bytes":          int64(len(testFileContent)),
		"file_mtime":          int64(srcStats.ModTime().Unix()),
		"resumable_upload_id": "testResumableUploadId",
	}
	if !reflect.DeepEqual(wantTaskResParams, msg.TaskResParams) {
		t.Errorf("taskResParams want: %+v, got: %+v", wantTaskResParams, msg.TaskResParams)
	}
}

// For testing purposes, this fakes an os.FileInfo object (which is the result
// of an os.File.Stat() call).
type fakeStats struct{}
func (f fakeStats) Name() string       { return "fake name" }
func (f fakeStats) Size() int64        { return 1234 }
func (f fakeStats) Mode() os.FileMode  { return os.FileMode(0) }
func (f fakeStats) ModTime() time.Time { return time.Unix(1234567890, 0) }
func (f fakeStats) IsDir() bool        { return false }
func (f fakeStats) Sys() interface{}   { return nil }

func TestPrepareResumableCopy(t *testing.T) {
	h := CopyHandler{}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// Verify the req method.
		if req.Method != "POST" {
			t.Error("want req method POST, got ", req.Method)
		}

		// Verify the req URL.
		var wantURL = []string{
			"https://www.googleapis.com/upload/storage/v1/",
			"b/dst_b/o",
			"alt=json",
			"ifGenerationMatch=77",
			"uploadType=resumable",
		}
		for _, w := range wantURL {
			if !strings.Contains(req.URL.String(), w) {
				t.Errorf("want URL contains %s, got %s", w, req.URL.String())
			}
		}

		// Verify the req headers.
		var wantHeaders = map[string][]string{
			"Content-Type":            {"application/json; charset=UTF-8"},
			"Content-Length":          {"87"},
			"User-Agent":              {userAgent},
			"X-Upload-Content-Length": {"1234"},
			"X-Upload-Content-Type":   {"text/plain; charset=utf-8"},
		}
		for wantKey, wantVal := range wantHeaders {
			headerVal, ok := req.Header[wantKey]
			if !ok {
				t.Errorf("want req header %s, not present", wantKey)
			} else if len(headerVal) != len(wantVal) {
				t.Errorf("for header %s want %v, got %v", wantKey, wantVal, headerVal)
			} else {
				for i := range wantVal {
					if headerVal[i] != wantVal[i] {
						t.Errorf("header %s want val %v, got %v", wantKey, wantVal[i], headerVal[i])
					}
				}
			}
		}

		// Verify the req body
		var o raw.Object
		err := json.NewDecoder(req.Body).Decode(&o)
		if err != nil {
			t.Error("couldn't decode req.Body for testing, err:", err)
		}
		if o.Name != "dst_o" {
			t.Errorf("want object name dst_o, got %s", o.Name)
		}
		if o.Bucket != "dst_b" {
			t.Errorf("want object bucket dst_b, got %s", o.Bucket)
		}
		if modtime, ok := o.Metadata[dcp.MTIME_ATTR_NAME]; !ok || modtime != "1234567890" {
			t.Errorf("want object metadata mtime 12345890, got %v", modtime)
		}

		// Make a fake resposne to carry on.
		res := &http.Response{
			StatusCode: 200,
			Header:     make(map[string][]string),
		}
		res.Header.Add("Location", "testResumableUploadId")
		return res, nil
	}

	ctx := context.Background()
	c := &copyTaskSpec{"src_f", "dst_b", "dst_o", 77, 0, 0, 0, 0, 0, 10 /*bytesToCopy*/, ""}
	taskResParams := make(taskResParams)

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	resumableUploadId, err := h.prepareResumableCopy(ctx, c, taskResParams, srcFile, stats)
	if err != nil {
		t.Error("got ", err)
	}
	if resumableUploadId != "testResumableUploadId" {
		t.Error("want resumableUploadId testResumableUploadId, got ", resumableUploadId)
	}

	// Verify the task response.
	var wantTaskResParams = []struct {
		key string
		val interface{}
	}{
		{"file_bytes", int64(1234)},
		{"file_mtime", int64(1234567890)},
		{"resumable_upload_id", "testResumableUploadId"},
	}
	for _, wtr := range wantTaskResParams {
		var val interface{}
		var ok bool
		if val, ok = taskResParams[wtr.key]; !ok {
			t.Errorf("want taskResParams key %v to exist, it didn't", wtr.key)
		}
		if val != wtr.val {
			t.Errorf("want taskResParams %s %[2]v/%[2]T, got %[3]v/%[3]T", wtr.key, wtr.val, val)
		}
	}
}

func TestCopyResumableChunkFinal(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		object := &raw.Object{
			Name:    "dst_o",
			Bucket:  "dst_b",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "modTime",
		}
		body := new(bytes.Buffer)
		_ = json.NewEncoder(body).Encode(object)
		res := &http.Response{
			StatusCode: 200,
			Header:     make(map[string][]string),
			Body:       ioutil.NopCloser(body),
		}
		return res, nil
	}

	ctx := context.Background()
	c := &copyTaskSpec{"src_f", "dst_b", "dst_o", 77, 0, 0, 0, 0, 0, 100 /*bytesToCopy*/, "ruID"}

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	taskResParams := make(taskResParams)
	logEntry := dcp.LogEntry{}

	err = h.copyResumableChunk(ctx, c, taskResParams, srcFile, stats, logEntry)
	if err != nil {
		t.Error("got ", err)
	}

	// Verify task parameter updates.
	val, ok := taskResParams["bytes_copied"]
	if !ok {
		t.Error("want taskResParams key bytes_copied to exist, it didn't")
	}
	if val != int64(len(testFileContent)) {
		t.Errorf("want taskResParams bytes_copied %[1]v/%[1]T, got %[2]v/%[2]T", int64(len(testFileContent)), val)
	}

	// Verify task logEntry.
	var wantLogEntry = []struct {
		key string
		val interface{}
	}{
		{"dst_crc32c", int64(testCRC32C)},
		{"dst_bytes", uint64(len(testFileContent))},
		{"dst_modified_time", "modTime"},
		{"bytes_copied", int64(len(testFileContent))},
	}
	for _, wle := range wantLogEntry {
		var val interface{}
		var ok bool
		if val, ok = logEntry[wle.key]; !ok {
			t.Errorf("want logEntry key %v to exist, it didn't", wle.key)
		}
		if val != wle.val {
			t.Errorf("want logEntry %s %[2]v/%[2]T, got %[3]v/%[3]T", wle.key, wle.val, val)
		}
	}
}

func TestCopyResumableChunkNotFinal(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		object := &raw.Object{
			Name:    "dst_o",
			Bucket:  "dst_b",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "modTime",
		}
		body := new(bytes.Buffer)
		_ = json.NewEncoder(body).Encode(object)
		res := &http.Response{
			StatusCode: 200,
			Header:     make(map[string][]string),
			Body:       ioutil.NopCloser(body),
		}
		return res, nil
	}

	ctx := context.Background()
	c := &copyTaskSpec{"src_f", "dst_b", "dst_o", 77, 0, 0, 0, 0, 0, 10 /*bytesToCopy*/, "ruID"}

	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	taskResParams := make(taskResParams)
	logEntry := dcp.LogEntry{}

	err = h.copyResumableChunk(ctx, c, taskResParams, srcFile, stats, logEntry)
	if err != nil {
		t.Error("got ", err)
	}

	// Verify the task response.
	var wantTaskResParams = []struct {
		key string
		val interface{}
	}{
		{"crc32c", int64(testTenByteCRC32C)},
		{"bytes_copied", int64(10)},
	}
	for _, wtr := range wantTaskResParams {
		var val interface{}
		var ok bool
		if val, ok = taskResParams[wtr.key]; !ok {
			t.Errorf("want taskResParams key %v to exist, it didn't", wtr.key)
		}
		if val != wtr.val {
			t.Errorf("want taskResParams %s %[2]v/%[2]T, got %[3]v/%[3]T", wtr.key, wtr.val, val)
		}
	}

	// Verify task logEntry.
	val, ok := logEntry["bytes_copied"]
	if !ok {
		t.Error("want logEntry key bytes_copied to exist, it didn't")
	}
	if val != int64(10) {
		t.Errorf("want logEntry bytes_copied %[1]v/%[1]T, got %[2]v/%[2]T", int64(10), val)
	}
}

func TestResumedCopyRequest(t *testing.T) {
	h := CopyHandler{}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// Verify method and URL.
		if req.Method != "PUT" {
			t.Error("want req method PUT, got ", req.Method)
		}
		if req.URL.String() != "testURL" {
			t.Error("want req URL testURL, got ", req.URL.String())
		}

		// Copy all the reqeust headers to the response so we can test
		// them outside this httpDoFunc.
		res := &http.Response{
			Header: make(map[string][]string),
		}
		for k, v := range req.Header {
			res.Header.Add(k, v[0])
		}
		return res, nil
	}

	ctx := context.Background()
	data := bytes.NewBufferString("0123456789")
	// Test a variety of final/offset/size combinations to verify the
	// Content-Range and Content-Length header values.
	var testCases = []struct {
		final             bool
		offset            int64
		size              int64
		wantContentRange  string
		wantContentLength string
	}{
		// A three chunk transfer.
		{false, 0, 4, "bytes 0-3/*", "4"},
		{false, 4, 4, "bytes 4-7/*", "4"},
		{true, 8, 2, "bytes 8-9/10", "2"},

		// Transfer all remaining bytes.
		{true, 0, 0, "bytes */0", "0"},
		{true, 5, 0, "bytes */5", "0"},
	}
	for _, tc := range testCases {
		res, err := h.resumedCopyRequest(ctx, "testURL", data, tc.offset, tc.size, tc.final)
		if err != nil {
			t.Errorf("want err nil, got %v", err)
		}
		var wantHeaders = map[string][]string{
			"Content-Range":      {tc.wantContentRange},
			"Content-Length":     {tc.wantContentLength},
			"X-Guploader-No-308": {"yes"},
		}
		for wantKey, wantVal := range wantHeaders {
			headerVal, ok := res.Header[wantKey]
			if !ok {
				t.Errorf("want header %s, not present in %+v", wantKey, res.Header)
			} else if len(headerVal) != len(wantVal) {
				t.Errorf("for header %s want %v, got %v", wantKey, wantVal, headerVal)
			} else {
				for i := range wantVal {
					if headerVal[i] != wantVal[i] {
						t.Errorf("header %s want val %v, got %v", wantKey, wantVal[i], headerVal[i])
					}
				}
			}
		}
	}
}

func TestStatusResumeIncomplete(t *testing.T) {
	if statusResumeIncomplete(nil) != false {
		t.Errorf("want false, got true")
	}
	res := &http.Response{
		Header: make(map[string][]string),
	}
	res.Header.Add("X-Http-Status-Code-Override", "308")
	if statusResumeIncomplete(res) != true {
		t.Errorf("want true, got false")
	}
}

func TestCodecUint32(t *testing.T) {
	for _, u := range []uint32{0, 1, 256, 0xFFFFFFFF} {
		s := encodeUint32(u)
		d, err := decodeUint32(s)
		if err != nil {
			t.Fatal(err)
		}
		if d != u {
			t.Errorf("got %d, want input %d", d, u)
		}
	}
}
