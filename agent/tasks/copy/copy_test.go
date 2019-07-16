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

package copy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"golang.org/x/sync/semaphore"
	raw "google.golang.org/api/storage/v1"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const (
	testFileContent   = "Ephemeral test file content for copy_test.go."
	testCRC32C        = 3923584507 // CRC32C of testFileContent.
	testTenByteCRC32C = 1069694901 // CRC32C of the first 10-bytes of testFileContent.
)

func testCopySpec(expGenNum, ccSize int64, ruID string) *taskpb.Spec {
	*copyChunkSize = int(ccSize)
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
	h := CopyHandler{concurrentCopySem: semaphore.NewWeighted(1)}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = "file does not exist"
	taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())
	if isValid, errMsg := common.IsValidFailureMsg("task", taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskRespMsg); !isValid {
		t.Error(errMsg)
	}
}

func TestCRC32CMismtach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := common.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C: 12345, // Incorrect CRC32C.
	})

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{
		gcs:               mockGCS,
		concurrentCopySem: semaphore.NewWeighted(1),
	}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())
	if isValid, errMsg := common.IsValidFailureMsg("task", taskpb.FailureType_HASH_MISMATCH_FAILURE, taskRespMsg); !isValid {
		t.Error(errMsg)
	}
}

func TestCopyEntireFileSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsModTime := time.Now()
	writer := common.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C:  uint32(testCRC32C),
		Size:    int64(len(testFileContent)),
		Updated: gcsModTime,
	})

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{
		gcs:               mockGCS,
		concurrentCopySem: semaphore.NewWeighted(1),
	}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())
	if isValid, errMsg := common.IsValidSuccessMsg("task", taskRespMsg); !isValid {
		t.Error(errMsg)
	}
	if writer.WrittenString() != testFileContent {
		t.Errorf("written string want \"%s\", got \"%s\"",
			testFileContent, writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLog := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{
				SrcFile:   tmpFile,
				SrcBytes:  int64(len(testFileContent)),
				SrcMTime:  srcStats.ModTime().Unix(),
				SrcCrc32C: testCRC32C,

				DstFile:   "bucket/object",
				DstBytes:  int64(len(testFileContent)),
				DstMTime:  gcsModTime.Unix(),
				DstCrc32C: testCRC32C,

				BytesCopied: int64(len(testFileContent)),
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestCopyEntireFileEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsModTime := time.Now()
	writer := common.NewStringWriteCloser(&storage.ObjectAttrs{
		CRC32C:  uint32(0),
		Size:    int64(0),
		Updated: gcsModTime,
	})

	tmpFile := common.CreateTmpFile("", "test-agent", "")
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{
		gcs:               mockGCS,
		concurrentCopySem: semaphore.NewWeighted(1),
	}
	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())
	if isValid, errMsg := common.IsValidSuccessMsg("task", taskRespMsg); !isValid {
		t.Error(errMsg)
	}
	if writer.WrittenString() != "" {
		t.Errorf("written string want \"%s\", got \"%s\"",
			"", writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLog := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{
				SrcFile:  tmpFile,
				SrcMTime: srcStats.ModTime().Unix(),

				DstFile:  "bucket/object",
				DstMTime: gcsModTime.Unix(),
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestCopyBundle(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsModTime := time.Now()

	type bundledFileTestData struct {
		fileName string
		fileData string
		crc      uint32
		size     int64
		bucket   string
		object   string

		wantStatus  taskpb.Status
		wantFailure taskpb.FailureType
		wantLog     *taskpb.CopyLog
	}

	tests := []struct {
		desc          string
		bundledFiles  []*bundledFileTestData
		bundleStatus  taskpb.Status
		bundleFailure taskpb.FailureType
		bundleLog     *taskpb.CopyBundleLog
	}{
		{
			desc: "test file bundle success",
			bundledFiles: []*bundledFileTestData{
				&bundledFileTestData{
					fileName:   "test-file-1",
					fileData:   "File 1 data content",
					crc:        0x3CAC94DC,
					size:       19,
					bucket:     "bucket",
					object:     "object1",
					wantStatus: taskpb.Status_SUCCESS,
				},
				&bundledFileTestData{
					fileName:   "test-file-2",
					fileData:   "The data of File 2",
					crc:        0xACDDCFCD,
					size:       18,
					bucket:     "bucket",
					object:     "object2",
					wantStatus: taskpb.Status_SUCCESS,
				},
			},
			bundleStatus: taskpb.Status_SUCCESS,
			bundleLog: &taskpb.CopyBundleLog{
				FilesCopied: 2,
				BytesCopied: 37,
			},
		},
		{
			desc: "test partial files success",
			bundledFiles: []*bundledFileTestData{
				&bundledFileTestData{
					fileName:    "test-file-1",
					fileData:    "File 1 data content",
					crc:         0x12345678, // This is invalid CRC to simulate failure.
					size:        19,
					bucket:      "bucket",
					object:      "object1",
					wantStatus:  taskpb.Status_FAILED,
					wantFailure: taskpb.FailureType_HASH_MISMATCH_FAILURE,
				},
				&bundledFileTestData{
					fileName:   "test-file-2",
					fileData:   "The data of File 2",
					crc:        0xACDDCFCD,
					size:       18,
					bucket:     "bucket",
					object:     "object2",
					wantStatus: taskpb.Status_SUCCESS,
				},
			},
			bundleStatus:  taskpb.Status_FAILED,
			bundleFailure: taskpb.FailureType_UNKNOWN_FAILURE,
			bundleLog: &taskpb.CopyBundleLog{
				FilesCopied: 1,
				BytesCopied: 18,
				FilesFailed: 1,
				BytesFailed: 19,
			},
		},
	}
	for _, tc := range tests {
		mockGCS := gcloud.NewMockGCS(mockCtrl)
		bundleSpec := &taskpb.CopyBundleSpec{}
		type fileInfo struct {
			crc    uint32
			size   int64
			writer *common.StringWriteCloser
		}
		objectMap := make(map[string]fileInfo)

		writerFunc := func(ctx context.Context, bucket, object string, cond interface{}) *common.StringWriteCloser {
			mapIndex := fmt.Sprintf("%s/%s", bucket, object)
			fileCrc := objectMap[mapIndex].crc
			fileSize := objectMap[mapIndex].size
			filewriter := common.NewStringWriteCloser(&storage.ObjectAttrs{
				CRC32C:  fileCrc,
				Size:    fileSize,
				Updated: gcsModTime,
			})

			objectMap[mapIndex] = fileInfo{
				crc:    fileCrc,
				size:   fileSize,
				writer: filewriter,
			}
			return filewriter
		}

		for _, file := range tc.bundledFiles {
			objectMap[fmt.Sprintf("%s/%s", file.bucket, file.object)] = fileInfo{
				crc:    file.crc,
				size:   file.size,
				writer: nil,
			}
			file.fileName = common.CreateTmpFile("", file.fileName, file.fileData)
			defer os.Remove(file.fileName)

			mockGCS.EXPECT().NewWriterWithCondition(context.Background(), file.bucket, file.object, gomock.Any()).DoAndReturn(writerFunc)
			bundleSpec.BundledFiles = append(bundleSpec.BundledFiles, &taskpb.BundledFile{
				CopySpec: &taskpb.CopySpec{
					DstBucket: file.bucket,
					DstObject: file.object,
					SrcFile:   file.fileName,
				},
			})
		}

		h := CopyHandler{
			gcs:               mockGCS,
			concurrentCopySem: semaphore.NewWeighted(1),
		}
		taskReqMsg := &taskpb.TaskReqMsg{
			TaskRelRsrcName: "task",
			Spec:            &taskpb.Spec{Spec: &taskpb.Spec_CopyBundleSpec{bundleSpec}},
		}
		taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())

		// Check for the overall task status
		t.Logf("CopyHandler.Do(%q)", tc.desc)
		if tc.bundleStatus == taskpb.Status_SUCCESS {
			if isValid, errMsg := common.IsValidSuccessMsg("task", taskRespMsg); !isValid {
				t.Error(errMsg)
			}
		} else {
			if isValid, errMsg := common.IsValidFailureMsg("task", tc.bundleFailure, taskRespMsg); !isValid {
				t.Error(errMsg)
			}
		}

		// Check for the overall bundle log.
		wantLog := &taskpb.Log{Log: &taskpb.Log_CopyBundleLog{tc.bundleLog}}
		if !proto.Equal(taskRespMsg.Log, wantLog) {
			t.Errorf("CopyHandler.Do(%q), got log = %+v, want: %+v", tc.desc, taskRespMsg.Log, wantLog)
		}

		// Check the status of each of the written file.
		resBundledFiles := taskRespMsg.RespSpec.GetCopyBundleSpec().BundledFiles
		for i, file := range tc.bundledFiles {
			mapIndex := fmt.Sprintf("%s/%s", file.bucket, file.object)
			if objectMap[mapIndex].writer.WrittenString() != file.fileData {
				t.Errorf("CopyHandler.Do(%q), written string: %q, want %q", tc.desc, objectMap[mapIndex].writer.WrittenString(), file.fileData)
			}
			srcStats, _ := os.Stat(file.fileName)
			wantLog := &taskpb.CopyLog{
				SrcFile:   file.fileName,
				SrcBytes:  file.size,
				SrcMTime:  srcStats.ModTime().Unix(),
				SrcCrc32C: file.crc,

				DstFile:   fmt.Sprintf("%s/%s", file.bucket, file.object),
				DstBytes:  file.size,
				DstMTime:  gcsModTime.Unix(),
				DstCrc32C: file.crc,

				BytesCopied: file.size,
			}

			if resBundledFiles[i].Status != file.wantStatus {
				t.Errorf("CopyHandler.Do(%q), got status: %s, want: %s", tc.desc, resBundledFiles[i].Status, file.wantStatus)
			}
			if file.wantStatus == taskpb.Status_SUCCESS {
				if !proto.Equal(resBundledFiles[i].CopyLog, wantLog) {
					t.Errorf("CopyHandler.Do(%q), file %d log = %+v, want: %+v", tc.desc, i, resBundledFiles[i].CopyLog, wantLog)
				}
			}
			if resBundledFiles[i].FailureType != file.wantFailure {
				t.Errorf("CopyHandler.Do(%q), got failureType: %s, want: %s", tc.desc, resBundledFiles[i].FailureType, file.wantFailure)
			}
		}
	}
}

func TestCopyHandlerDoResumable(t *testing.T) {
	h := CopyHandler{concurrentCopySem: semaphore.NewWeighted(1)}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// Read the http.Request.Body to invoke the CRC32UpdatingReader.
		buf := make([]byte, 1024)
		var err error
		for err == nil {
			_, err = req.Body.Read(buf)
		}

		// This bogus response serves both the prepareResumableCopy and
		// copyResumableChunk requests.
		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "2012-11-01T22:08:41+00:00",
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

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)

	taskReqMsg := testCopyTaskReqMsg()
	taskReqMsg.Spec.GetCopySpec().SrcFile = tmpFile
	*copyChunkSize = 10
	*copyEntireFileLimit = 10
	taskRespMsg := h.Do(context.Background(), taskReqMsg, time.Now())
	if isValid, errMsg := common.IsValidSuccessMsg("task", taskRespMsg); !isValid {
		t.Error(errMsg)
	}

	srcStats, _ := os.Stat(tmpFile)
	wantLog := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{
				SrcFile:  tmpFile,
				SrcBytes: int64(len(testFileContent)),
				SrcMTime: srcStats.ModTime().Unix(),

				DstFile: "bucket/object",

				BytesCopied: 10,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}

	wantTaskRespSpec := testCopyTaskReqMsg().Spec
	wantTaskRespSpec.GetCopySpec().SrcFile = tmpFile
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
func (f fakeStats) Size() int64        { return 45 }
func (f fakeStats) Mode() os.FileMode  { return os.FileMode(0) }
func (f fakeStats) ModTime() time.Time { return time.Unix(0, 1234567890000000000) }
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
			"X-Upload-Content-Length": {"45"},
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
	copySpec := testCopySpec(77, 10, "").GetCopySpec()

	wantRespCopySpec := proto.Clone(copySpec).(*taskpb.CopySpec)
	wantRespCopySpec.FileBytes = 45
	wantRespCopySpec.FileMTime = 1234567890
	wantRespCopySpec.ResumableUploadId = "testResumableUploadId"

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	if err := h.prepareResumableCopy(ctx, copySpec, srcFile, stats); err != nil {
		t.Error("got ", err)
	}
	if !proto.Equal(copySpec, wantRespCopySpec) {
		t.Errorf("copySpec = %v, want: %v", copySpec, wantRespCopySpec)
	}
}

func TestCopyResumableChunkFinal(t *testing.T) {
	h := CopyHandler{}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// Read the http.Request.Body to invoke the CRC32UpdatingReader.
		buf := make([]byte, 1024)
		var err error
		for err == nil {
			_, err = req.Body.Read(buf)
		}

		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "2012-11-01T22:08:41+00:00",
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
	copySpec := testCopySpec(77, 100, "ruID").GetCopySpec()

	wantRespCopySpec := proto.Clone(copySpec).(*taskpb.CopySpec)
	wantRespCopySpec.BytesCopied = int64(len(testFileContent))

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	log := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{},
		},
	}
	err = h.copyResumableChunk(ctx, copySpec, srcFile, stats, log.GetCopyLog())
	if err != nil {
		t.Error("got ", err)
	}

	if !proto.Equal(copySpec, wantRespCopySpec) {
		t.Errorf("copySpec = %v, want: %v", copySpec, wantRespCopySpec)
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{
				SrcCrc32C: testCRC32C,

				DstBytes:    int64(len(testFileContent)),
				DstCrc32C:   testCRC32C,
				DstMTime:    1351807721,
				BytesCopied: int64(len(testFileContent)),
			},
		},
	}
	if !proto.Equal(log, wantLog) {
		t.Errorf("log = %+v, want: %+v", log, wantLog)
	}
}

func TestCopyResumableChunkNotFinal(t *testing.T) {
	h := CopyHandler{}
	h.httpDoFunc = func(ctx context.Context, h *http.Client, req *http.Request) (*http.Response, error) {
		// Read the http.Request.Body to invoke the CRC32UpdatingReader.
		buf := make([]byte, 1024)
		var err error
		for err == nil {
			_, err = req.Body.Read(buf)
		}

		object := &raw.Object{
			Name:    "object",
			Bucket:  "bucket",
			Crc32c:  encodeUint32(testCRC32C),
			Size:    uint64(len(testFileContent)),
			Updated: "2012-11-01T22:08:41+00:00",
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
	copySpec := testCopySpec(77, 10, "ruID").GetCopySpec()

	wantRespCopySpec := proto.Clone(copySpec).(*taskpb.CopySpec)
	wantRespCopySpec.BytesCopied = 10
	wantRespCopySpec.Crc32C = testTenByteCRC32C

	tmpFile := common.CreateTmpFile("", "test-agent", testFileContent)
	defer os.Remove(tmpFile)
	srcFile, err := os.Open(tmpFile)
	if err != nil {
		t.Error("Couldn't open testing srcFile, err: ", err)
	}
	defer srcFile.Close()
	var stats fakeStats

	log := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{},
		},
	}
	err = h.copyResumableChunk(ctx, copySpec, srcFile, stats, log.GetCopyLog())
	if err != nil {
		t.Error("got ", err)
	}

	if !proto.Equal(copySpec, wantRespCopySpec) {
		t.Errorf("copySpec = %v, want: %v", copySpec, wantRespCopySpec)
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_CopyLog{
			CopyLog: &taskpb.CopyLog{BytesCopied: 10},
		},
	}
	if !proto.Equal(log, wantLog) {
		t.Errorf("log = %+v, want: %+v", log, wantLog)
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

func TestShouldRetry(t *testing.T) {
	testCases := []struct {
		status int
		err    error
		want   bool
	}{
		{status: 200, want: false},
		{status: 308, want: false},
		{status: 403, want: false},
		{status: 408, want: true},
		{status: 429, want: true},
		{status: 500, want: true},
		{status: 503, want: true},
		{status: 600, want: false},
		{err: io.EOF, want: false},
		{err: errors.New("random badness"), want: false},
		{err: io.ErrUnexpectedEOF, want: true},
		{err: &net.AddrError{}, want: false},              // Not temporary.
		{err: &net.DNSError{IsTimeout: true}, want: true}, // Temporary.
	}
	for _, tt := range testCases {
		if got := shouldRetry(tt.status, tt.err); got != tt.want {
			t.Errorf("shouldRetry(%d, %v) = %t; want %t", tt.status, tt.err, got, tt.want)
		}
	}
}

func TestGetBundleLogAndError(t *testing.T) {
	testCases := []struct {
		desc            string
		copyBundleSpec  *taskpb.CopyBundleSpec
		wantStatus      taskpb.Status
		wantFailureType taskpb.FailureType
		wantLog         *taskpb.CopyBundleLog
	}{
		{
			desc: "All success",
			copyBundleSpec: &taskpb.CopyBundleSpec{
				BundledFiles: []*taskpb.BundledFile{
					&taskpb.BundledFile{
						Status: taskpb.Status_SUCCESS,
						CopyLog: &taskpb.CopyLog{
							BytesCopied: 1,
						},
						CopySpec: &taskpb.CopySpec{
							SrcFile: testFileContent,
						},
					},
				},
			},
			wantStatus: taskpb.Status_SUCCESS,
			wantLog: &taskpb.CopyBundleLog{
				FilesCopied: 1,
				BytesCopied: 1,
			},
		},
		{
			desc: "Service-induced error only",
			copyBundleSpec: &taskpb.CopyBundleSpec{
				BundledFiles: []*taskpb.BundledFile{
					&taskpb.BundledFile{
						Status:         taskpb.Status_FAILED,
						FailureType:    taskpb.FailureType_HASH_MISMATCH_FAILURE,
						FailureMessage: taskpb.FailureType_HASH_MISMATCH_FAILURE.String(),
						CopyLog: &taskpb.CopyLog{
							SrcBytes: 1,
						},
						CopySpec: &taskpb.CopySpec{
							SrcFile: testFileContent,
						},
					},
				},
			},
			wantStatus:      taskpb.Status_FAILED,
			wantFailureType: taskpb.FailureType_UNKNOWN_FAILURE,
			wantLog: &taskpb.CopyBundleLog{
				FilesFailed: 1,
				BytesFailed: 1,
			},
		},
		{
			desc: "Not service-induced error only",
			copyBundleSpec: &taskpb.CopyBundleSpec{
				BundledFiles: []*taskpb.BundledFile{
					&taskpb.BundledFile{
						Status:         taskpb.Status_FAILED,
						FailureType:    taskpb.FailureType_FILE_NOT_FOUND_FAILURE,
						FailureMessage: taskpb.FailureType_FILE_NOT_FOUND_FAILURE.String(),
						CopyLog: &taskpb.CopyLog{
							SrcBytes: 1,
						},
						CopySpec: &taskpb.CopySpec{
							SrcFile: testFileContent,
						},
					},
				},
			},
			wantStatus:      taskpb.Status_FAILED,
			wantFailureType: taskpb.FailureType_NOT_SERVICE_INDUCED_UNKNOWN_FAILURE,
			wantLog: &taskpb.CopyBundleLog{
				FilesFailed: 1,
				BytesFailed: 1,
			},
		},
		{
			desc: "Both service and not service induced errors",
			copyBundleSpec: &taskpb.CopyBundleSpec{
				BundledFiles: []*taskpb.BundledFile{
					&taskpb.BundledFile{
						Status:         taskpb.Status_FAILED,
						FailureType:    taskpb.FailureType_FILE_NOT_FOUND_FAILURE,
						FailureMessage: taskpb.FailureType_FILE_NOT_FOUND_FAILURE.String(),
						CopyLog: &taskpb.CopyLog{
							SrcBytes: 1,
						},
						CopySpec: &taskpb.CopySpec{
							SrcFile: testFileContent,
						},
					},
					&taskpb.BundledFile{
						Status:         taskpb.Status_FAILED,
						FailureType:    taskpb.FailureType_HASH_MISMATCH_FAILURE,
						FailureMessage: taskpb.FailureType_HASH_MISMATCH_FAILURE.String(),
						CopyLog: &taskpb.CopyLog{
							SrcBytes: 2,
						},
						CopySpec: &taskpb.CopySpec{
							SrcFile: testFileContent,
						},
					},
				},
			},
			wantStatus:      taskpb.Status_FAILED,
			wantFailureType: taskpb.FailureType_UNKNOWN_FAILURE,
			wantLog: &taskpb.CopyBundleLog{
				FilesFailed: 2,
				BytesFailed: 3,
			},
		},
	}
	for _, tc := range testCases {
		log, err := getBundleLogAndError(tc.copyBundleSpec)
		if err != nil {
			if log.BytesFailed != tc.wantLog.BytesFailed {
				t.Errorf("test case: %s of getBundleLogAndError, got bytesfailed: %+v, want: %+v", tc.desc, log.BytesFailed, tc.wantLog.BytesFailed)
			}
			if log.FilesFailed != tc.wantLog.FilesFailed {
				t.Errorf("test case: %s of getBundleLogAndError, got filesfailed: %+v, want: %+v", tc.desc, log.FilesFailed, tc.wantLog.FilesFailed)
			}
			failureType := common.GetFailureTypeFromError(err)
			if failureType != tc.wantFailureType {
				t.Errorf("test case: %s of getBundleLogAndError, got failureType: %+v, want: %+v", tc.desc, failureType, tc.wantFailureType)
			}
		}
	}
}

func TestShouldDoTimeAwareCopy(t *testing.T) {
	testCases := []struct {
		desc        string
		copySpec    *taskpb.CopySpec
		reqStartDur time.Duration
		jrRRN       string
		want        bool
	}{
		{
			"No resumable ID",
			&taskpb.CopySpec{
				ResumableUploadId: "",
				BytesCopied:       5,
				FileBytes:         10,
			},
			5*time.Second - *copyWorkDuration,
			"jrRRN_test",
			false,
		},
		{
			"No bytes left to copy",
			&taskpb.CopySpec{
				ResumableUploadId: "id",
				BytesCopied:       10,
				FileBytes:         10,
			},
			5*time.Second - *copyWorkDuration,
			"jrRRN_test",
			false,
		},
		{
			"Not enough time",
			&taskpb.CopySpec{
				ResumableUploadId: "id",
				BytesCopied:       5,
				FileBytes:         10,
			},
			-5*time.Second - *copyWorkDuration,
			"jrRRN_test",
			false,
		},
		{
			"Paused job run",
			&taskpb.CopySpec{
				ResumableUploadId: "id",
				BytesCopied:       5,
				FileBytes:         10,
			},
			5*time.Second - *copyWorkDuration,
			"Bogus job run name, not in jobBWS map.",
			false,
		},
		{
			"Success",
			&taskpb.CopySpec{
				ResumableUploadId: "id",
				BytesCopied:       5,
				FileBytes:         10,
			},
			5*time.Second - *copyWorkDuration,
			"jrRRN_test",
			true,
		},
	}
	for _, tc := range testCases {
		// Set the active/paused jobs.
		jobBWs := []*controlpb.JobRunBandwidth{
			&controlpb.JobRunBandwidth{
				JobrunRelRsrcName: "jrRRN_test",
				Bandwidth:         10 * 1024 * 1024,
			},
		}
		rate.ProcessJobRunBandwidths(jobBWs, nil)

		reqStart := time.Now().Add(tc.reqStartDur)
		if got := shouldDoTimeAwareCopy(tc.copySpec, reqStart, tc.jrRRN); got != tc.want {
			t.Errorf("%v: shouldDoTimeAwareCopy(...) got %v, want %v", tc.desc, got, tc.want)
		}
	}
}
