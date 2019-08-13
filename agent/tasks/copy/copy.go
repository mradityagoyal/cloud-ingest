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
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/gensupport"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	raw "google.golang.org/api/storage/v1"
	htransport "google.golang.org/api/transport/http"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const (
	userAgent         = "google-cloud-ingest-on-premises-agent TransferService/1.0 (GPN:transferservice_onpremnfs; Data moved from onpremnfs to GCS)"
	userAgentInternal = "google-cloud-ingest-on-premises-agent"
	MTIME_ATTR_NAME   = "goog-reserved-file-mtime"
)

var (
	internalTesting     = flag.Bool("internal-testing", false, "Agent running for Google internal testing purposes.")
	copyFilesPerCPU     = flag.Int("copy-files-per-cpu", 8, "Files to copy (per CPU) in parallel. Can be overridden by setting copy-files.")
	copyFiles           = flag.Int("copy-files", 0, "Files to copy in parallel. If > 0 this will override copy-files-per-cpu.")
	fileReadBuf         = flag.Int("file-read-buf", 1*1024*1024, "Read buffer size for each concurrent file copy. Increasing this raises Agent memory usage, but decreases potential reads to the source file system.")
	copyChunkSize       = flag.Int("copy-chunk-size", 128*1024*1024, "The amount of bytes to send in a single HTTP request.")
	copyEntireFileLimit = flag.Int("copy-entire-file-limit", 8*1024*1024, "Copy a file in a single HTTP request if it's below this size.")
	copyWorkDuration    = flag.Duration("copy-work-duration", 1*time.Minute, "The amount of time to spend copying a single file.")
)

// NewResumableHttpClient creates a new http.Client suitable for resumable copies.
func NewResumableHttpClient(ctx context.Context, opts ...option.ClientOption) (*http.Client, error) {
	userAgentStr := userAgent
	if *internalTesting {
		userAgentStr = userAgentInternal
	}
	// TODO(b/74008724): We likely don't need full control, only read and write. Limit this.
	o := []option.ClientOption{
		option.WithScopes(raw.DevstorageFullControlScope),
		option.WithUserAgent(userAgentStr),
	}
	opts = append(o, opts...)
	hc, _, err := htransport.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create resumable HTTP client, err: %v", err)
	}
	return hc, nil
}

// CopyHandler is responsible for handling copy tasks.
type CopyHandler struct {
	gcs               gcloud.GCS
	hc                *http.Client
	concurrentCopySem *semaphore.Weighted // Limits the number of concurrent goroutines uploading files.
	statsTracker      *stats.Tracker      // For tracking bytes sent/copied.

	// Exposed here only for testing purposes.
	httpDoFunc func(context.Context, *http.Client, *http.Request) (*http.Response, error)
}

// NewCopyHandler creates a CopyHandler with storage.Client and http.Client.
func NewCopyHandler(storageClient *storage.Client, hc *http.Client, st *stats.Tracker) *CopyHandler {
	cf := *copyFiles
	if cf <= 0 {
		cf = *copyFilesPerCPU * runtime.NumCPU()
	}
	glog.Info("CopyHandler initialized with copy-files:", cf)
	return &CopyHandler{
		gcs:               gcloud.NewGCSClient(storageClient),
		hc:                hc,
		concurrentCopySem: semaphore.NewWeighted(int64(cf)),
		httpDoFunc:        ctxhttp.Do,
		statsTracker:      st,
	}
}

func checkResumableFileStats(c *taskpb.CopySpec, fileinfo os.FileInfo) error {
	if c.FileBytes != fileinfo.Size() {
		return common.AgentError{
			Msg: fmt.Sprintf(
				"File size changed during the copy. Expected:%+v, got:%+v",
				c.FileBytes, fileinfo.Size()),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	if c.FileMTime != fileinfo.ModTime().Unix() {
		return common.AgentError{
			Msg: fmt.Sprintf(
				"File mtime changed during the copy. Expected:%+v, got:%+v",
				c.FileMTime, fileinfo.ModTime().Unix()),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	return nil
}

func (h *CopyHandler) checkFileStats(beforeStats os.FileInfo, f *os.File) error {
	statStart := time.Now()
	afterStats, err := f.Stat()
	h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyStatMs: stats.DurMs(statStart)})
	if err != nil {
		return err
	}
	if beforeStats.Size() != afterStats.Size() || beforeStats.ModTime() != afterStats.ModTime() {
		return common.AgentError{
			Msg: fmt.Sprintf(
				"File stats changed during the copy. Before stats:%+v, after stats: %+v",
				beforeStats, afterStats),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	return nil
}

func (h *CopyHandler) parseBucketAndObjectName(src string) (bucket string, object string, err error) {
	s := strings.TrimPrefix(src, "gs://")
	spl := strings.SplitN(s, "/", 2)
	if len(spl) != 2 {
		return "", "", fmt.Errorf("couldn't parse bucket/object from string %s", src)
	}
	return spl[0], spl[1], nil
}

func (h *CopyHandler) handleDownloadSpec(ctx context.Context, copySpec *taskpb.CopySpec) (*taskpb.CopyLog, error) {
	cl := &taskpb.CopyLog{
		SrcFile: copySpec.SrcFile,
		DstFile: copySpec.DstObject,
	}

	bn, on, err := h.parseBucketAndObjectName(copySpec.SrcFile)
	if err != nil {
		return cl, err
	}

	// Challenge: how to ensure two agents writing simultaneously to a file don't
	// conflict with each other?
	// Proposed solution: assume it is always safe to write the same bytes to
	// the same offset.  Without this we need some global lock service.
	// Challenge: if the generation number changes, how to clean up old files?
	// Proposed solution: temporary file contains generation number. When worker
	// notices that generation in spec does not match generation in GCS, it
	// attempts to delete the old file, and must finish the deletion
	// attempt before responding on Pub/Sub.
	// As a safeguard, check generation number at the end of the copied chunk
	// as opposed to the beginning, and always clean up before replying?

	// Open the cloud object metadata and read the generation number
	// If the temporary file does not exist OR we are starting at offset 0
	// (Re)create the temporary file with the correct total size
	// Seek to the correct offset in the temporary file.
	// Download the desired range to the temporary file.
	// If the file is done:
	// Confirm the E2E checksum we calculated on the download.
	// Rename the file to its final name

	// Quick impl:
	//   Create temporary file with generation number on start.
	//   Write chunks.
	//   Check generation number at end of each chunk.
	//   Rename at the end.

	resumedCopy, err := checkCopyDownloadTaskSpec(copySpec)
	if err != nil {
		return cl, err
	}

	if resumedCopy {
		//glog.Infof("download: resumed copy, bucket %s object %s", bn, on)
	} else {
		//glog.Infof("download: new copy, bucket %s object %s", bn, on)
	}

	oa, err := h.gcs.GetAttrs(ctx, bn, on)
	//glog.Infof("download: got object attrs")
	if err != nil {
		return cl, err
	}
	genStr := strconv.FormatInt(oa.Generation, 10)
	if err != nil {
		return cl, err
	}
	//glog.Infof("download: got generation")

	// Temporary filename is destination.generationnumber
	var fb strings.Builder
	fb.WriteString(copySpec.DstObject)
	fb.WriteString(".")
	fb.WriteString(genStr)
	fn := fb.String()

	var f *os.File
	if !resumedCopy {
		err := os.MkdirAll(path.Dir(fn), 0777)
		if err != nil {
			return cl, err
		}

		f, err = os.Create(fn)
		//glog.Infof("download: Created file %s", fn)
		if err != nil {
			return cl, err
		}

		defer f.Close()
		if err = os.Truncate(fn, oa.Size); err != nil {
			return cl, err
		}
		//glog.Infof("download: Truncated file %s", fn)
	} else {
		f, err = os.OpenFile(fn, os.O_WRONLY, 0666)
		if err != nil {
			return cl, err
		}
		defer f.Close()
	}

	// TODO(thobrla): consolidate bytesToCopy instances
	//glog.Infof("setting bytes from copyspec")
	bytesToCopy := int64(1 << 25)
	if copySpec.FileBytes-copySpec.BytesCopied <= bytesToCopy {
		// TODO(thobrla): respect size sent by DCP
		bytesToCopy = copySpec.FileBytes - copySpec.BytesCopied
		//bytesToCopy = oa.Size - copySpec.BytesCopied
	}
	//glog.Infof("creating gcs reader for %s bytestoCopy %d", on, bytesToCopy)
	srcReader, err := h.gcs.NewRangeReader(ctx, bn, on, copySpec.BytesCopied, bytesToCopy)
	if err != nil {
		//glog.Infof("gcs reader err %v", err)

		return cl, err
	}

	//log.Infof("created gcs reader")
	err = h.downloadChunk(ctx, copySpec, srcReader, oa, f, fn, cl, bytesToCopy)
	if err != nil {
		return cl, err
	}

	// This populates the log entry for the audit logs and for tracking
	// bytes. Bytes are only counted when the task moves to "success", so
	// there won't be any double counting.
	cl.SrcBytes = oa.Size
	cl.SrcMTime = oa.Created.Unix()

	// TODO: We've written data.  Confirm generation number hasn't changed.
	return cl, nil
}

func (h *CopyHandler) downloadChunk(ctx context.Context, c *taskpb.CopySpec, srcReader io.Reader,
	srcAttrs *storage.ObjectAttrs, dstFile *os.File, tempName string, cl *taskpb.CopyLog, bytesToCopy int64) error {
	final := false
	if bytesToCopy <= 0 || bytesToCopy+c.BytesCopied >= srcAttrs.Size {
		// c.BytesToCopy <= 0 indicates that the rest of the file should be copied.
		bytesToCopy = srcAttrs.Size - c.BytesCopied
		final = true
	}

	//glog.Infof("download: downloading chunk")

	pos, err := dstFile.Seek(c.BytesCopied, 0)
	if err != nil {
		return err
	}
	if pos != c.BytesCopied {
		glog.Errorf("Seek expected pos %d, got %d", c.BytesCopied, pos)
	}
	// TODO(thobrla) TODO: timing-aware download
	// TODO(thobrla) TODO: bytes tracking reader
	// var r io.Reader = io.LimitReader(srcReader, bytesToCopy) // Wrap the srcFile in a LimitReader.
	r := h.statsTracker.NewCopyByteTrackingReader(srcReader) // Wrap the srcReader with a CopyByteTrackingReader.
	r = io.LimitReader(r, bytesToCopy)                       // Wrap the srcFile in a LimitReader
	r = rate.NewRateLimitingReader(r)                        // Wrap with a RateLimitingReader.
	srcCRC32C := c.Crc32C                                    // Set the initial crc32.
	r = NewCRC32UpdatingReader(r, &srcCRC32C)                // Wrap with a CRC32UpdatingReader.
	tr := stats.NewTimingReader(r)                           // Wrap with a TimingReader.
	w := bufio.NewWriterSize(dstFile, *fileReadBuf)          // Wrap with a buffered writer.

	// TODO(thobrla): Fix up stats tracker to calculate timing for downloads and then record pulse stats
	written, err := io.Copy(w, tr)
	if err != nil {
		return err
	}

	// Ensure any remaining buffered bytes are written.
	if err = w.Flush(); err != nil {
		return err
	}
	//glog.Infof("downloadChunk: Wrote %d bytes at offset %d", written, pos)

	if int64(written) != bytesToCopy {
		return fmt.Errorf("file %s wrote %d bytes, expected %d", c.DstObject, written, bytesToCopy)
	}

	c.BytesCopied = bytesToCopy + c.BytesCopied

	if final {
		err = dstFile.Close()
		if err != nil {
			return err
		}
		//glog.Infof("downloadChunk: finalized %s --> %s", tempName, c.DstObject)
		if err := os.Rename(tempName, c.DstObject); err != nil {
			return err
		}
	} else {
		// TODO(thobrla): Stop trying to trick the DCP into doing a "resumable" download
		c.ResumableUploadId = "placeholder"
	}

	return nil
}

func (h *CopyHandler) handleCopySpec(ctx context.Context, copySpec *taskpb.CopySpec) (*taskpb.CopyLog, error) {
	if strings.HasPrefix(copySpec.SrcFile, "gs://") {
		return h.handleDownloadSpec(ctx, copySpec)
	}

	cl := &taskpb.CopyLog{
		SrcFile: copySpec.SrcFile,
		DstFile: path.Join(copySpec.DstBucket, copySpec.DstObject),
	}

	resumedCopy, err := checkCopyTaskSpec(copySpec)
	if err != nil {
		return cl, err
	}

	// Open the on-premises file, and check the file stats if necessary.
	openStart := time.Now()
	srcFile, err := os.Open(copySpec.SrcFile)
	h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyOpenMs: stats.DurMs(openStart)})
	if err != nil {
		return cl, err
	}
	defer srcFile.Close()

	statStart := time.Now()
	fileinfo, err := srcFile.Stat()
	h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyStatMs: stats.DurMs(statStart)})
	if err != nil {
		return cl, err
	}
	// This populates the log entry for the audit logs and for tracking
	// bytes. Bytes are only counted when the task moves to "success", so
	// there won't be any double counting.
	cl.SrcBytes = fileinfo.Size()
	cl.SrcMTime = fileinfo.ModTime().Unix()
	if resumedCopy {
		// TODO(b/74009003): When implementing "synchronization" rethink how
		// the file stat parameters are set and compared.
		if err = checkResumableFileStats(copySpec, fileinfo); err != nil {
			return cl, err
		}
	}

	// Copy the entire file or start a resumable copy.
	if !resumedCopy {
		// Start a copy. If the file is small enough copy the entire file, otherwise begin a resumable copy.
		if fileinfo.Size() <= int64(*copyEntireFileLimit) || *copyChunkSize <= 0 {
			err = h.copyEntireFile(ctx, copySpec, srcFile, fileinfo, cl)
			if err != nil {
				return cl, err
			}
		} else {
			if err := h.prepareResumableCopy(ctx, copySpec, srcFile, fileinfo); err != nil {
				return cl, err
			}
			resumedCopy = true
		}
	}
	if resumedCopy {
		err = h.copyResumableChunk(ctx, copySpec, srcFile, fileinfo, cl)
		if err != nil {
			return cl, err
		}
	}

	// Now that data has been sent, check that the fileinfo stats haven't changed.
	if err = h.checkFileStats(fileinfo, srcFile); err != nil {
		return cl, err
	}

	return cl, nil
}

func isServiceInducedError(failureType taskpb.FailureType) bool {
	switch failureType {
	case taskpb.FailureType_UNKNOWN_FAILURE, taskpb.FailureType_HASH_MISMATCH_FAILURE:
		return true
	default:
		return false
	}
}

func getBundleLogAndError(bs *taskpb.CopyBundleSpec) (*taskpb.CopyBundleLog, error) {
	var log taskpb.CopyBundleLog
	var atLeastOneServiceInducedError bool
	for _, bf := range bs.BundledFiles {
		if bf.Status == taskpb.Status_SUCCESS {
			log.FilesCopied++
			log.BytesCopied += bf.CopyLog.BytesCopied
		} else {
			if !atLeastOneServiceInducedError && isServiceInducedError(bf.FailureType) {
				atLeastOneServiceInducedError = true
			}
			log.FilesFailed++
			log.BytesFailed += bf.CopyLog.SrcBytes
			glog.Warningf("bundledFile %v, failed with err: %v", bf.CopySpec.SrcFile, bf.FailureMessage)
		}
	}
	var err error
	if log.FilesFailed > 0 {
		failureType := taskpb.FailureType_NOT_SERVICE_INDUCED_UNKNOWN_FAILURE
		if atLeastOneServiceInducedError {
			failureType = taskpb.FailureType_UNKNOWN_FAILURE
		}
		err = common.AgentError{
			Msg:         fmt.Sprintf("CopyBundle had %v failures", log.FilesFailed),
			FailureType: failureType,
		}
	}
	return &log, err
}

func shouldDoTimeAwareCopy(copySpec *taskpb.CopySpec, reqStart time.Time, jobRunRelRsrcName string) bool {
	// Do a time aware copy iteration iff
	// 1. The copy is resuamble.
	// 2. There are bytes left to copy.
	// 3. We haven't exceeded the work duration.
	// 4. The JobRun is active (not paused).
	return copySpec.ResumableUploadId != "" && copySpec.BytesCopied < copySpec.FileBytes && time.Now().Before(reqStart.Add(*copyWorkDuration)) && rate.IsJobRunActive(jobRunRelRsrcName)
}

func (h *CopyHandler) handleCopySpecTimeAware(ctx context.Context, copySpec *taskpb.CopySpec, reqStart time.Time, jobRunRelRsrcName string) (*taskpb.CopySpec, *taskpb.CopyLog, error) {
	// Perform the initial copy.
	copyLog, err := h.handleCopySpec(ctx, copySpec) // Updates 'copySpec' in place.
	if err != nil {
		return copySpec, copyLog, err
	}
	// If the file copy is resumable and timing allows then continue working on the copy.
	for shouldDoTimeAwareCopy(copySpec, reqStart, jobRunRelRsrcName) {
		goodSpec := proto.Clone(copySpec).(*taskpb.CopySpec)
		goodCopyLog := proto.Clone(copyLog).(*taskpb.CopyLog)
		copyLog, err = h.handleCopySpec(ctx, copySpec)
		if err != nil {
			// If we have a previously good state just return that.
			return goodSpec, goodCopyLog, nil
		}
	}
	return copySpec, copyLog, err
}

func (h *CopyHandler) handleCopyBundleSpec(ctx context.Context, bundleSpec *taskpb.CopyBundleSpec, reqStart time.Time, jobRunRelRsrcName string) (*taskpb.CopyBundleLog, error) {
	var wg sync.WaitGroup
	for _, bf := range bundleSpec.BundledFiles {
		wg.Add(1)
		go func(bf *taskpb.BundledFile) {
			if len(bundleSpec.BundledFiles) > 1 {
				// Apply concurrency limiting to copy tasks with multiple bundled files.
				h.concurrentCopySem.Acquire(ctx, 1)
				defer h.concurrentCopySem.Release(1)
			}
			defer wg.Done()
			var err error
			bf.CopySpec, bf.CopyLog, err = h.handleCopySpecTimeAware(ctx, bf.CopySpec, reqStart, jobRunRelRsrcName)
			bf.FailureType = common.GetFailureTypeFromError(err)
			bf.FailureMessage = fmt.Sprint(err)
			if err == nil {
				bf.Status = taskpb.Status_SUCCESS
			} else {
				bf.Status = taskpb.Status_FAILED
			}
		}(bf)
	}
	wg.Wait()
	return getBundleLogAndError(bundleSpec)
}

func (h *CopyHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg, reqStart time.Time) *taskpb.TaskRespMsg {
	var respSpec *taskpb.Spec
	var log *taskpb.Log
	var err error

	if taskReqMsg.Spec.GetCopySpec() != nil {
		var cl *taskpb.CopyLog
		copySpec := proto.Clone(taskReqMsg.Spec.GetCopySpec()).(*taskpb.CopySpec)
		copySpec, cl, err = h.handleCopySpecTimeAware(ctx, copySpec, reqStart, taskReqMsg.JobrunRelRsrcName)
		respSpec = &taskpb.Spec{Spec: &taskpb.Spec_CopySpec{copySpec}}
		log = &taskpb.Log{Log: &taskpb.Log_CopyLog{cl}}
	} else if taskReqMsg.Spec.GetCopyBundleSpec() != nil {
		var cbl *taskpb.CopyBundleLog
		bundleSpec := proto.Clone(taskReqMsg.Spec.GetCopyBundleSpec()).(*taskpb.CopyBundleSpec)
		cbl, err = h.handleCopyBundleSpec(ctx, bundleSpec, reqStart, taskReqMsg.JobrunRelRsrcName)
		respSpec = &taskpb.Spec{Spec: &taskpb.Spec_CopyBundleSpec{bundleSpec}}
		log = &taskpb.Log{Log: &taskpb.Log_CopyBundleLog{cbl}}
	} else {
		err = errors.New("CopyHandler.Do taskReqMsg.Spec is neither CopySpec nor CopyBundleSpec")
	}

	return common.BuildTaskRespMsg(taskReqMsg, respSpec, log, err)
}

func (h *CopyHandler) copyEntireFile(ctx context.Context, c *taskpb.CopySpec, srcFile *os.File, fileinfo os.FileInfo, cl *taskpb.CopyLog) error {
	w := h.gcs.NewWriterWithCondition(ctx, c.DstBucket, c.DstObject, common.GetGCSGenerationNumCondition(c.ExpectedGenerationNum))
	if t, ok := w.(*storage.Writer); ok {
		t.Metadata = map[string]string{MTIME_ATTR_NAME: strconv.FormatInt(fileinfo.ModTime().Unix(), 10)}
	}

	var srcCRC32C uint32
	r := h.statsTracker.NewCopyByteTrackingReader(srcFile) // Wrap the srcFile with a CopyByteTrackingReader.
	r = rate.NewRateLimitingReader(r)                      // Wrap with a RateLimitingReader.
	r = NewCRC32UpdatingReader(r, &srcCRC32C)              // Wrap with a CRC32UpdatingReader.
	tr := stats.NewTimingReader(r)                         // Wrap with a TimingReader.

	// Copy the file using io.Copy. This allocates a small temp buffer and handles the Read+Write calls.
	writeStart := time.Now()
	_, err := io.Copy(w, tr)
	h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyWriteMs: stats.DurMs(writeStart.Add(tr.ReadDur()))})
	if err != nil {
		w.CloseWithError(err)
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Record some attributes.
	dstAttrs := w.Attrs()
	cl.DstBytes = dstAttrs.Size
	cl.DstCrc32C = dstAttrs.CRC32C
	cl.DstMTime = dstAttrs.Updated.Unix()
	cl.SrcCrc32C = srcCRC32C
	cl.DstMd5 = base64.StdEncoding.EncodeToString(dstAttrs.MD5)
	cl.BytesCopied = fileinfo.Size()

	// Verify the CRC32C.
	if dstAttrs.CRC32C != srcCRC32C {
		return common.AgentError{
			Msg: fmt.Sprintf("CRC32C mismatch for file %s (%d) against object %s (%d)",
				c.SrcFile, srcCRC32C, c.DstObject, dstAttrs.CRC32C),
			FailureType: taskpb.FailureType_HASH_MISMATCH_FAILURE,
		}
	}

	return nil
}

func contentType(srcFile io.Reader) string {
	// 512 is the max needed by http.DetectContentType, see:
	// https://golang.org/pkg/net/http/#DetectContentType
	sniffBuf, err := ioutil.ReadAll(io.LimitReader(srcFile, 512))
	if err != nil {
		return "application/octet-stream"
	}
	return http.DetectContentType(sniffBuf)
}

// prepareResumableCopy makes a request to GCS to begin a resumable copy. It
// updates the copy spec (with the resuambleUploadId and other file metadata)
// which will be sent to the DCP for future work on this resumable copy task.
func (h *CopyHandler) prepareResumableCopy(ctx context.Context, c *taskpb.CopySpec, srcFile io.Reader, fileinfo os.FileInfo) error {
	// Create the request URL.
	urlParams := make(gensupport.URLParams)
	urlParams.Set("ifGenerationMatch", fmt.Sprint(c.ExpectedGenerationNum))
	urlParams.Set("alt", "json")
	urlParams.Set("uploadType", "resumable")
	url := googleapi.ResolveRelative("https://www.googleapis.com/upload/storage/v1/", "b/{bucket}/o")
	url += "?" + urlParams.Encode()

	// Create the request body.
	object := &raw.Object{
		Name:   c.DstObject,
		Bucket: c.DstBucket,
		Metadata: map[string]string{
			MTIME_ATTR_NAME: strconv.FormatInt(fileinfo.ModTime().Unix(), 10),
		},
	}
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(object); err != nil {
		return fmt.Errorf("json.NewEncoder(body).Encode(object) err: %v", err)
	}

	userAgentStr := userAgent
	if *internalTesting {
		userAgentStr = userAgentInternal
	}

	// Create the request headers.
	reqHeaders := make(http.Header)
	reqHeaders.Set("Content-Type", "application/json; charset=UTF-8")
	reqHeaders.Set("Content-Length", fmt.Sprint(body.Len()))
	reqHeaders.Set("User-Agent", userAgentStr)
	reqHeaders.Set("X-Upload-Content-Length", fmt.Sprint(fileinfo.Size()))
	reqHeaders.Set("X-Upload-Content-Type", contentType(srcFile))

	// Stitch all the pieces together into an HTTP request.
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("http.NewRequest err: %v", err)
	}
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{"bucket": c.DstBucket})

	// Send the HTTP request!
	resp, err := h.httpDoFunc(ctx, h.hc, req)
	if err != nil {
		return fmt.Errorf("httpDoFunc err: %v", err)
	}
	defer googleapi.CloseBody(resp)
	if err = googleapi.CheckResponse(resp); err != nil {
		return err
	}

	// This function was successful, update the copy spec.
	c.FileBytes = fileinfo.Size()
	c.FileMTime = fileinfo.ModTime().Unix()
	// TODO(b/74009190): Consider renaming this, or somehow indicating that
	// this is a full URL. The Agent needs to be aware that this is a full
	// URL, however the DCP really only cares that this is some sort of ID.
	c.ResumableUploadId = resp.Header.Get("Location")

	return nil
}

// copyResumableChunk sends a chunk of the srcFile to GCS as part of a resumable
// copy task. This function also updates the CopySpec and CopyLog, both of
// which are sent to the DCP.
func (h *CopyHandler) copyResumableChunk(ctx context.Context, c *taskpb.CopySpec, srcFile *os.File, fileinfo os.FileInfo, cl *taskpb.CopyLog) error {
	final := false
	bytesToCopy := int64(*copyChunkSize)
	if bytesToCopy <= 0 || bytesToCopy+c.BytesCopied >= fileinfo.Size() {
		// bytesToCopy <= 0 indicates that the rest of the file should be copied.
		bytesToCopy = fileinfo.Size() - c.BytesCopied
		final = true
	}

	var srcCRC32C uint32

	// This loop will retry multiple times if the HTTP response returns a retryable error.
	var backoff BackOff
	var delay time.Duration
	var resp *http.Response
	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		seekStart := time.Now()
		_, err = srcFile.Seek(c.BytesCopied, 0)
		h.statsTracker.RecordPulseStats(&stats.PulseStats{CopySeekMs: stats.DurMs(seekStart)})
		if err != nil {
			return err
		}
		r := h.statsTracker.NewCopyByteTrackingReader(srcFile) // Wrap the srcFile in a CopyByteTrackingReader.
		r = io.LimitReader(r, bytesToCopy)                     // Wrap with a LimitReader.
		r = NewSemAcquiringReader(r, ctx)                      // Wrap with a SemAcquiringReader.
		r = bufio.NewReaderSize(r, *fileReadBuf)               // Wrap with a buffered reader.
		r = rate.NewRateLimitingReader(r)                      // Wrap with a RateLimitingReader.
		srcCRC32C = c.Crc32C                                   // Set the initial crc32.
		r = NewCRC32UpdatingReader(r, &srcCRC32C)              // Wrap with a CRC32UpdatingReader.
		tr := stats.NewTimingReader(r)                         // Wrap with a TimingReader.

		// Perform the copy!
		writeStart := time.Now()
		resp, err = h.resumedCopyRequest(ctx, c.ResumableUploadId, tr, c.BytesCopied, int64(bytesToCopy), final)
		h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyWriteMs: stats.DurMs(writeStart.Add(tr.ReadDur()))})

		var status int
		if resp != nil {
			status = resp.StatusCode
		}

		// Check if we should retry the request.
		if shouldRetry(status, err) {
			h.statsTracker.RecordPulseStats(&stats.PulseStats{CopyInternalRetries: 1})
			var retry bool
			if delay, retry = backoff.GetDelay(); retry {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				continue
			}
		}
		break
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("resumedCopyRequest err: %v", err)
	}
	if resp.StatusCode == 410 {
		return common.AgentError{
			Msg: fmt.Sprintf("GCS HTTP 410 for file %s, uploadid %v", c.SrcFile, c.ResumableUploadId),
			FailureType: taskpb.FailureType_GCS_RESUMABLE_ID_GONE_FAILURE,
		}
	}
	if err = googleapi.CheckResponse(resp); err != nil {
		return err
	}

	if final {
		if statusResumeIncomplete(resp) {
			return errors.New("statusResumeIncomplete was true for final copy")
		}

		// Parse the object from the http response.
		obj := &raw.Object{}
		if err = gensupport.DecodeResponse(&obj, resp); err != nil {
			return fmt.Errorf("gensupport.DecodeResponse err: %v", err)
		}
		var dstCRC32C uint32
		if dstCRC32C, err = decodeUint32(obj.Crc32c); err != nil {
			return fmt.Errorf("decodeUint32 err: %v", err)
		}

		// Check the CRC32C.
		if dstCRC32C != srcCRC32C {
			return common.AgentError{
				Msg: fmt.Sprintf("CRC32C mismatch for file %s (%d) against object %s (%d)",
					c.SrcFile, srcCRC32C, c.DstObject, dstCRC32C),
				FailureType: taskpb.FailureType_HASH_MISMATCH_FAILURE,
			}
		}
		cl.DstCrc32C = dstCRC32C
		cl.DstMd5 = obj.Md5Hash
		cl.DstBytes = int64(obj.Size)
		var t time.Time
		if err := t.UnmarshalText([]byte(obj.Updated)); err != nil {
			return fmt.Errorf("t.UnmarshalText err: %v", err)
		}
		cl.DstMTime = t.Unix()
		cl.SrcCrc32C = srcCRC32C
	} else {
		c.Crc32C = srcCRC32C
	}
	c.BytesCopied += int64(bytesToCopy)
	cl.BytesCopied = c.BytesCopied

	return nil
}

func (h *CopyHandler) resumedCopyRequest(ctx context.Context, URL string, data io.Reader, offset, size int64, final bool) (*http.Response, error) {
	req, err := http.NewRequest("PUT", URL, data)
	if err != nil {
		return nil, err
	}

	req.ContentLength = size
	var contentRange string
	if final {
		if size == 0 {
			contentRange = fmt.Sprintf("bytes */%d", offset)
		} else {
			contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, offset+size-1, offset+size)
		}
	} else {
		contentRange = fmt.Sprintf("bytes %d-%d/*", offset, offset+size-1)
	}
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Content-Length", fmt.Sprint(size))

	// Google's upload endpoint uses status code 308 for a
	// different purpose than the "308 Permanent Redirect"
	// since-standardized in RFC 7238. Because of the conflict in
	// semantics, Google added this new request header which
	// causes it to not use "308" and instead reply with 200 OK
	// and sets the upload-specific "X-HTTP-Status-Code-Override:
	// 308" response header.
	req.Header.Set("X-Guploader-No-308", "yes")

	return h.httpDoFunc(ctx, h.hc, req)
}

func statusResumeIncomplete(resp *http.Response) bool {
	// This is how the server signals "status resume incomplete"
	// when X-Guploader-No-308 is set to "yes":
	return resp != nil && resp.Header.Get("X-Http-Status-Code-Override") == "308"
}

// Decode a uint32 encoded in Base64 in big-endian byte order.
func decodeUint32(b64 string) (uint32, error) {
	d, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, err
	}
	if len(d) != 4 {
		return 0, fmt.Errorf("storage: %q does not encode a 32-bit value", d)
	}
	return uint32(d[0])<<24 + uint32(d[1])<<16 + uint32(d[2])<<8 + uint32(d[3]), nil
}

// Encode a uint32 as Base64 in big-endian byte order.
func encodeUint32(u uint32) string {
	b := []byte{byte(u >> 24), byte(u >> 16), byte(u >> 8), byte(u)}
	return base64.StdEncoding.EncodeToString(b)
}

// shouldRetry returns true if the HTTP response / error indicates that the
// request should be attempted again.
func shouldRetry(status int, err error) bool {
	if 500 <= status && status <= 599 {
		return true
	}
	if status == 408 {
		return true
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	if err == io.ErrUnexpectedEOF {
		return true
	}
	if err, ok := err.(net.Error); ok {
		return err.Temporary()
	}
	return false
}
