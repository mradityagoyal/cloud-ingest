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
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"golang.org/x/sync/semaphore"
	raw "google.golang.org/api/storage/v1"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const (
	testFileContent   = "Ephemeral test file content for copy_test.go."
	testCRC32C        = 3923584507 // CRC32C of testFileContent.
	testTenByteCRC32C = 1069694901 // CRC32C of the first 10-bytes of testFileContent.
)

func testCopySpec(expGenNum, bytesToCopy int64, ruID string) *taskpb.Spec {
	return &taskpb.Spec{
		Spec: &taskpb.Spec_CopySpec{
			CopySpec: &taskpb.CopySpec{
				DstBucket:             "bucket",
				DstObject:             "object",
				SrcFile:               "file",
				ExpectedGenerationNum: expGenNum,
				FileBytes:             0,
				FileMTime:             0,
				BytesCopied:           0,
				BytesToCopy:           bytesToCopy,
				ResumableUploadId:     ruID,
				Crc32C:                0,
			},
		},
	}
}

func testCopyTaskReqMsg() *taskpb.TaskReqMsg {
	return &taskpb.TaskReqMsg{
		TaskRelRsrcName: "task",
		Spec:            testCopySpec(0, 0, ""),
	}
}

func TestSourceNotFound(t *testing.T) {
	h := CopyHandler{}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = "file does not exist"
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkFailureWithType("task", taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskRespMsg, t)
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
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkFailureWithType("task", taskpb.FailureType_UNKNOWN_FAILURE, taskRespMsg, t)
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
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkFailureWithType("task", taskpb.FailureType_HASH_MISMATCH_FAILURE, taskRespMsg, t)
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
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkSuccessMsg("task", taskRespMsg, t)
	if writer.WrittenString() != testFileContent {
		t.Errorf("written string want \"%s\", got \"%s\"",
			testFileContent, writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLogFields := LogFields{
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
	if wantLogFields.String() != taskRespMsg.AgentLogFields {
		t.Errorf("log entry want: %+v, got: %+v", wantLogFields, taskRespMsg.AgentLogFields)
	}
}

func TestCopyEntireFileEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsModTime := time.Now()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C:  uint32(0),
		Size:    int64(0),
		Updated: gcsModTime,
	})

	tmpFile := helpers.CreateTmpFile("", "test-agent", "")
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	copyMemoryLimit = defaultCopyMemoryLimit
	h := CopyHandler{mockGCS, 5, nil, semaphore.NewWeighted(copyMemoryLimit), nil}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkSuccessMsg("task", taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("written string want \"%s\", got \"%s\"",
			"", writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLogFields := LogFields{
		"worker_id":         workerID,
		"src_crc32c":        uint32(0),
		"dst_crc32c":        uint32(0),
		"src_bytes":         int64(0),
		"dst_bytes":         int64(0),
		"src_file":          tmpFile,
		"dst_file":          "bucket/object",
		"src_modified_time": srcStats.ModTime(),
		"dst_modified_time": gcsModTime,
	}
	if wantLogFields.String() != taskRespMsg.AgentLogFields {
		t.Errorf("log entry want: %+v, got: %+v", wantLogFields, taskRespMsg.AgentLogFields)
	}

}

func TestCopyHanderDoResumable(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// This bogus response serves both the prepareResumableCopy and
		// copyResumableChunk requests.
		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
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

	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskReqMsg.Spec.GetCopySpec().BytesToCopy = 10
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	checkSuccessMsg("task", taskRespMsg, t)

	srcStats, _ := os.Stat(tmpFile)
	wantLogFields := LogFields{
		"bytes_copied":      int64(10),
		"dst_file":          "bucket/object",
		"src_bytes":         int64(len(testFileContent)),
		"src_file":          tmpFile,
		"src_modified_time": srcStats.ModTime(),
		"worker_id":         workerID,
	}
	if wantLogFields.String() != taskRespMsg.AgentLogFields {
		t.Errorf("log entry want: %+v, got: %+v", wantLogFields, taskRespMsg.AgentLogFields)
	}

	wantTaskRespSpec := testCopyTaskReqMsg().Spec
	wantTaskRespSpec.GetCopySpec().SrcFile = tmpFile
	wantTaskRespSpec.GetCopySpec().BytesToCopy = 10
	wantTaskRespSpec.GetCopySpec().BytesCopied = 10
	wantTaskRespSpec.GetCopySpec().Crc32C = testTenByteCRC32C
	wantTaskRespSpec.GetCopySpec().FileBytes = int64(len(testFileContent))
	wantTaskRespSpec.GetCopySpec().FileMTime = srcStats.ModTime().Unix()
	wantTaskRespSpec.GetCopySpec().ResumableUploadId = "testResumableUploadId"
	if !proto.Equal(wantTaskRespSpec, taskRespMsg.RespSpec) {
		t.Errorf("taskRespMsg.RespSpec = %v, want: %v", taskRespMsg.RespSpec, wantTaskRespSpec)
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
			"b/bucket/o",
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
			"Content-Length":          {"89"},
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
		if o.Name != "object" {
			t.Errorf("want object name object, got %s", o.Name)
		}
		if o.Bucket != "bucket" {
			t.Errorf("want object bucket bucket, got %s", o.Bucket)
		}
		if modtime, ok := o.Metadata[MTIME_ATTR_NAME]; !ok || modtime != "1234567890" {
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
	reqCopySpec := testCopySpec(77, 10, "").GetCopySpec()
	respCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	resumableUploadId, err := h.prepareResumableCopy(ctx, reqCopySpec, respCopySpec, srcFile, stats)
	if err != nil {
		t.Error("got ", err)
	}
	if resumableUploadId != "testResumableUploadId" {
		t.Error("want resumableUploadId testResumableUploadId, got ", resumableUploadId)
	}

	wantRespCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	wantRespCopySpec.FileBytes = 1234
	wantRespCopySpec.FileMTime = 1234567890
	wantRespCopySpec.ResumableUploadId = "testResumableUploadId"
	if !proto.Equal(respCopySpec, wantRespCopySpec) {
		t.Errorf("respCopySpec = %v, want: %v", respCopySpec, wantRespCopySpec)
	}
}

func TestCopyResumableChunkFinal(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
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
	reqCopySpec := testCopySpec(77, 100, "ruID").GetCopySpec()
	respCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	logFields := LogFields{}
	err = h.copyResumableChunk(ctx, reqCopySpec, respCopySpec, srcFile, stats, logFields)
	if err != nil {
		t.Error("got ", err)
	}

	wantRespCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	wantRespCopySpec.BytesCopied = int64(len(testFileContent))
	if !proto.Equal(respCopySpec, wantRespCopySpec) {
		t.Errorf("respCopySpec = %v, want: %v", respCopySpec, wantRespCopySpec)
	}

	// Verify task logFields.
	var wantLogFields = []struct {
		key string
		val interface{}
	}{
		{"dst_crc32c", int64(testCRC32C)},
		{"dst_bytes", uint64(len(testFileContent))},
		{"dst_modified_time", "modTime"},
		{"src_crc32c", uint32(testCRC32C)},
		{"bytes_copied", int64(len(testFileContent))},
	}
	for _, wle := range wantLogFields {
		var val interface{}
		var ok bool
		if val, ok = logFields[wle.key]; !ok {
			t.Errorf("want logFields key %v to exist, it didn't", wle.key)
		}
		if val != wle.val {
			t.Errorf("want logFields %s %[2]v/%[2]T, got %[3]v/%[3]T", wle.key, wle.val, val)
		}
	}
}

func TestCopyResumableChunkNotFinal(t *testing.T) {
	h := CopyHandler{memoryLimiter: semaphore.NewWeighted(copyMemoryLimit)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
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
	reqCopySpec := testCopySpec(77, 10, "ruID").GetCopySpec()
	respCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	tmpFile := helpers.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	logFields := LogFields{}

	err = h.copyResumableChunk(ctx, reqCopySpec, respCopySpec, srcFile, stats, logFields)
	if err != nil {
		t.Error("got ", err)
	}

	wantRespCopySpec := proto.Clone(reqCopySpec).(*taskpb.CopySpec)
	wantRespCopySpec.BytesCopied = 10
	wantRespCopySpec.Crc32C = testTenByteCRC32C
	if !proto.Equal(respCopySpec, wantRespCopySpec) {
		t.Errorf("respCopySpec = %v, want: %v", respCopySpec, wantRespCopySpec)
	}

	// Verify task logFields.
	val, ok := logFields["bytes_copied"]
	if !ok {
		t.Error("want logFields key bytes_copied to exist, it didn't")
	}
	if val != int64(10) {
		t.Errorf("want logFields bytes_copied %[1]v/%[1]T, got %[2]v/%[2]T", int64(10), val)
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
