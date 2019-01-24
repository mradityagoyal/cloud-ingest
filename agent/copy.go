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
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	crc32pkg "hash/crc32" // Alias to disambiguate from usage.
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/storage"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
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
	defaultCopyMemoryLimit int64  = 1 << 30 // Default memory limit is 1 GiB.
	userAgent                     = "google-cloud-ingest-on-premises-agent"
	userAgentInternal             = "google-cloud-ingest-on-premises-agent"
	MTIME_ATTR_NAME        string = "goog-reserved-file-mtime"

	// Note: this default chunk size is only used if the DCP instructs the
	// Agent to copy the entire file but does not specify a chunk size. This
	// happens by sending a BytesToCopy value <= 0 in the CopyTaskSpec.
	veneerClientDefaultChunkSize = 1 << 27 // 128MiB.
)

var (
	copyMemoryLimit int64
	CRC32CTable     *crc32pkg.Table
	internalTesting bool
)

func init() {
	flag.Int64Var(&copyMemoryLimit, "copy-max-memory", defaultCopyMemoryLimit,
		"Max memory buffer (in bytes) consumed by the copy tasks.")
	flag.BoolVar(&internalTesting, "internal-testing", false,
		"Agent running for Google internal testing purposes.")
	CRC32CTable = crc32pkg.MakeTable(crc32pkg.Castagnoli)
}

// NewResumableHttpClient creates a new http.Client suitable for resumable copies.
func NewResumableHttpClient(ctx context.Context, opts ...option.ClientOption) (*http.Client, error) {
	userAgentStr := userAgent
	if internalTesting {
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
	gcs           gcloud.GCS
	hc            *http.Client
	memoryLimiter *semaphore.Weighted
	// concurrentCopySem is semaphore to limit the number of concurrent goroutines uploading files.
	concurrentCopySem *semaphore.Weighted

	statsTracker *stats.Tracker // For tracking bytes sent/copied.

	// Exposed here only for testing purposes.
	httpDoFunc func(context.Context, *http.Client, *http.Request) (*http.Response, error)
}

// NewCopyHandler creates a CopyHandler with storage.Client and http.Client.
// The maxParallelism is the max number of goroutines copying files concurrently.
func NewCopyHandler(storageClient *storage.Client, maxParallelism int, hc *http.Client, st *stats.Tracker) *CopyHandler {
	return &CopyHandler{
		gcs:               gcloud.NewGCSClient(storageClient),
		hc:                hc,
		memoryLimiter:     semaphore.NewWeighted(copyMemoryLimit),
		concurrentCopySem: semaphore.NewWeighted(int64(maxParallelism)),
		httpDoFunc:        ctxhttp.Do,
		statsTracker:      st,
	}
}

func checkResumableFileStats(c *taskpb.CopySpec, stats os.FileInfo) error {
	if c.FileBytes != stats.Size() {
		return AgentError{
			Msg: fmt.Sprintf(
				"File size changed during the copy. Expected:%+v, got:%+v",
				c.FileBytes, stats.Size()),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	if c.FileMTime != stats.ModTime().Unix() {
		return AgentError{
			Msg: fmt.Sprintf(
				"File mtime changed during the copy. Expected:%+v, got:%+v",
				c.FileMTime, stats.ModTime().Unix()),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	return nil
}

func checkFileStats(beforeStats os.FileInfo, f *os.File) error {
	afterStats, err := f.Stat()
	if err != nil {
		return err
	}
	if beforeStats.Size() != afterStats.Size() || beforeStats.ModTime() != afterStats.ModTime() {
		return AgentError{
			Msg: fmt.Sprintf(
				"File stats changed during the copy. Before stats:%+v, after stats: %+v",
				beforeStats, afterStats),
			FailureType: taskpb.FailureType_FILE_MODIFIED_FAILURE,
		}
	}
	return nil
}

func (h *CopyHandler) handleCopySpec(ctx context.Context, copySpec *taskpb.CopySpec) (*taskpb.CopyLog, error) {
	h.concurrentCopySem.Acquire(ctx, 1)
	defer h.concurrentCopySem.Release(1)
	cl := &taskpb.CopyLog{
		SrcFile: copySpec.SrcFile,
		DstFile: path.Join(copySpec.DstBucket, copySpec.DstObject),
	}

	resumedCopy, err := checkCopyTaskSpec(copySpec)
	if err != nil {
		return cl, err
	}

	// Open the on-premises file, and check the file stats if necessary.
	srcFile, err := os.Open(copySpec.SrcFile)
	if err != nil {
		return cl, err
	}
	defer srcFile.Close()
	stats, err := srcFile.Stat()
	if err != nil {
		return cl, err
	}
	// This populates the log entry for the audit logs and for tracking
	// bytes. Bytes are only counted when the task moves to "success", so
	// there won't be any double counting.
	cl.SrcBytes = stats.Size()
	cl.SrcMTime = stats.ModTime().Unix()
	if resumedCopy {
		// TODO(b/74009003): When implementing "synchronization" rethink how
		// the file stat parameters are set and compared.
		if err = checkResumableFileStats(copySpec, stats); err != nil {
			return cl, err
		}
	}

	// Copy the entire file or start a resumable copy.
	if !resumedCopy {
		// Start a copy. If the file is small enough (or BytesToCopy indicates so)
		// copy the entire file now. Otherwise, begin a resumable copy.
		if stats.Size() <= copySpec.BytesToCopy || copySpec.BytesToCopy <= 0 {
			err = h.copyEntireFile(ctx, copySpec, srcFile, stats, cl)
			if err != nil {
				return cl, err
			}
		} else {
			if err := h.prepareResumableCopy(ctx, copySpec, srcFile, stats); err != nil {
				return cl, err
			}
			resumedCopy = true
		}
	}
	if resumedCopy {
		err = h.copyResumableChunk(ctx, copySpec, srcFile, stats, cl)
		if err != nil {
			return cl, err
		}
	}

	// Now that data has been sent, check that the file stats haven't changed.
	if err = checkFileStats(stats, srcFile); err != nil {
		return cl, err
	}

	return cl, nil
}

func getBundleLogAndError(bs *taskpb.CopyBundleSpec) (*taskpb.CopyBundleLog, error) {
	var err error
	var log taskpb.CopyBundleLog
	for _, bf := range bs.BundledFiles {
		if bf.Status == taskpb.Status_SUCCESS {
			log.FilesCopied++
			log.BytesCopied += bf.CopyLog.BytesCopied
		} else {
			log.FilesFailed++
			log.BytesFailed += bf.CopyLog.SrcBytes
			if err == nil {
				err = AgentError{
					Msg:         "CopyBundle task failed, please check the spec for detailed per file error",
					FailureType: taskpb.FailureType_UNKNOWN_FAILURE,
				}
			}
		}
	}
	return &log, err
}

func (h *CopyHandler) handleCopyBundleSpec(ctx context.Context, bundleSpec *taskpb.CopyBundleSpec) (*taskpb.CopyBundleLog, error) {
	var wg sync.WaitGroup
	for _, bf := range bundleSpec.BundledFiles {
		wg.Add(1)
		go func(bf *taskpb.BundledFile) {
			defer wg.Done()
			var err error
			bf.CopyLog, err = h.handleCopySpec(ctx, bf.CopySpec)
			bf.FailureType = getFailureTypeFromError(err)
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

func (h *CopyHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	var respSpec *taskpb.Spec
	var log *taskpb.Log
	var err error

	if taskReqMsg.Spec.GetCopySpec() != nil {
		var cl *taskpb.CopyLog
		copySpec := proto.Clone(taskReqMsg.Spec.GetCopySpec()).(*taskpb.CopySpec)
		cl, err = h.handleCopySpec(ctx, copySpec)
		respSpec = &taskpb.Spec{Spec: &taskpb.Spec_CopySpec{copySpec}}
		log = &taskpb.Log{Log: &taskpb.Log_CopyLog{cl}}
	} else if taskReqMsg.Spec.GetCopyBundleSpec() != nil {
		var cbl *taskpb.CopyBundleLog
		bundleSpec := proto.Clone(taskReqMsg.Spec.GetCopyBundleSpec()).(*taskpb.CopyBundleSpec)
		cbl, err = h.handleCopyBundleSpec(ctx, bundleSpec)
		respSpec = &taskpb.Spec{Spec: &taskpb.Spec_CopyBundleSpec{bundleSpec}}
		log = &taskpb.Log{Log: &taskpb.Log_CopyBundleLog{cbl}}
	} else {
		err = errors.New("CopyHandler.Do taskReqMsg.Spec is neither CopySpec nor CopyBundleSpec")
	}

	return buildTaskRespMsg(taskReqMsg, respSpec, log, err)
}

func (h *CopyHandler) copyEntireFile(ctx context.Context, c *taskpb.CopySpec, srcFile *os.File, stats os.FileInfo, cl *taskpb.CopyLog) error {
	w := h.gcs.NewWriterWithCondition(ctx, c.DstBucket, c.DstObject,
		helpers.GetGCSGenerationNumCondition(c.ExpectedGenerationNum))

	bufSize := stats.Size()
	if t, ok := w.(*storage.Writer); ok {
		t.Metadata = map[string]string{
			MTIME_ATTR_NAME: strconv.FormatInt(stats.ModTime().Unix(), 10),
		}
		if c.BytesToCopy <= 0 {
			bufSize = veneerClientDefaultChunkSize
		}
		t.ChunkSize = int(bufSize)
	}

	// Create a buffer that respects the Agent's copyMemoryLimit.
	if bufSize > copyMemoryLimit {
		return fmt.Errorf(
			"memory buffer limit for copy tasks is %d bytes, but task requires %d bytes",
			copyMemoryLimit, bufSize)
	} else if bufSize < 1 {
		// Never allow a non-positive buf size (mainly for empty files).
		bufSize = 1
	}
	if err := h.memoryLimiter.Acquire(ctx, bufSize); err != nil {
		return err
	}
	defer h.memoryLimiter.Release(bufSize)
	buf := make([]byte, bufSize)

	// Wrap the srcFile with rate limiting and byte tracking readers.
	r := rate.NewRateLimitingReader(srcFile)
	if h.statsTracker != nil {
		r = h.statsTracker.NewByteTrackingReader(r)
	}

	// Perform the copy (by writing to the gcsWriter).
	var srcCRC32C uint32
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			w.CloseWithError(err)
			return err
		}
		_, err = w.Write(buf[:n])
		if err != nil {
			w.CloseWithError(err)
			return err
		}
		srcCRC32C = crc32pkg.Update(srcCRC32C, CRC32CTable, buf[:n])
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
	cl.BytesCopied = stats.Size()

	// Verify the CRC32C.
	if dstAttrs.CRC32C != srcCRC32C {
		return AgentError{
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
func (h *CopyHandler) prepareResumableCopy(ctx context.Context, c *taskpb.CopySpec, srcFile io.Reader, stats os.FileInfo) error {
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
			MTIME_ATTR_NAME: strconv.FormatInt(stats.ModTime().Unix(), 10),
		},
	}
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(object); err != nil {
		return fmt.Errorf("json.NewEncoder(body).Encode(object) err: %v", err)
	}

	userAgentStr := userAgent
	if internalTesting {
		userAgentStr = userAgentInternal
	}

	// Create the request headers.
	reqHeaders := make(http.Header)
	reqHeaders.Set("Content-Type", "application/json; charset=UTF-8")
	reqHeaders.Set("Content-Length", fmt.Sprint(body.Len()))
	reqHeaders.Set("User-Agent", userAgentStr)
	reqHeaders.Set("X-Upload-Content-Length", fmt.Sprint(stats.Size()))
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
	c.FileBytes = stats.Size()
	c.FileMTime = stats.ModTime().Unix()
	// TODO(b/74009190): Consider renaming this, or somehow indicating that
	// this is a full URL. The Agent needs to be aware that this is a full
	// URL, however the DCP really only cares that this is some sort of ID.
	c.ResumableUploadId = resp.Header.Get("Location")

	return nil
}

// copyResumableChunk sends a chunk of the srcFile to GCS as part of a resumable
// copy task. This function also updates the CopySpec and CopyLog, both of
// which are sent to the DCP.
func (h *CopyHandler) copyResumableChunk(ctx context.Context, c *taskpb.CopySpec, srcFile *os.File, stats os.FileInfo, cl *taskpb.CopyLog) error {
	bytesToCopy := c.BytesToCopy
	if bytesToCopy <= 0 {
		// c.BytesToCopy <= 0 indicates that the rest of the file should be copied.
		bytesToCopy = stats.Size() - c.BytesCopied
	}

	// Create a buffer that respects the Agent's copyMemoryLimit.
	if bytesToCopy > copyMemoryLimit {
		return fmt.Errorf(
			"total memory buffer limit for copy task is %d bytes, but task requires "+
				"%d bytes (resumeableChunkSize)",
			copyMemoryLimit, bytesToCopy)
	}
	if err := h.memoryLimiter.Acquire(ctx, bytesToCopy); err != nil {
		return fmt.Errorf("memoryLimiter.Acquire err: %v", err)
	}
	defer h.memoryLimiter.Release(bytesToCopy)
	buf := make([]byte, bytesToCopy)

	// Read the source file into the buffer from where the previous resumable-copy left off.
	srcFile.Seek(c.BytesCopied, 0)
	bytesRead := 0
	var err error
	for err == nil && bytesRead < int(bytesToCopy) {
		var n int
		n, err = srcFile.Read(buf[bytesRead:])
		bytesRead += n
	}
	buf = buf[:bytesRead]
	final := err == io.EOF
	if !final && err != nil {
		return fmt.Errorf("srcFile.Read non-io.EOF err: %v", err)
	}
	srcCRC32C := crc32pkg.Update(uint32(c.Crc32C), CRC32CTable, buf)

	// This loop will retry multiple times if the HTTP response returns a retryable error.
	var backoff BackOff
	var delay time.Duration
	var resp *http.Response
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Make a copy of the read bytes so that if we have to retry the send we don't have
		// to re-read from the srcFile (potentially hitting an on-premises NFS).
		copyBuf := make([]byte, len(buf))
		copy(copyBuf, buf)
		cbr := bytes.NewReader(copyBuf)

		// Wrap the chunk buffer with rate limiting and byte tracking readers.
		r := rate.NewRateLimitingReader(cbr)
		if h.statsTracker != nil {
			r = h.statsTracker.NewByteTrackingReader(r)
		}

		// Perform the copy!
		resp, err = h.resumedCopyRequest(ctx, c.ResumableUploadId, r, c.BytesCopied, int64(bytesRead), final)

		var status int
		if resp != nil {
			status = resp.StatusCode
		}

		// Check if we should retry the request.
		if shouldRetry(status, err) {
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
			return AgentError{
				Msg: fmt.Sprintf("CRC32C mismatch for file %s (%d) against object %s (%d)",
					c.SrcFile, srcCRC32C, c.DstObject, dstCRC32C),
				FailureType: taskpb.FailureType_HASH_MISMATCH_FAILURE,
			}
		}
		cl.DstCrc32C = dstCRC32C
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
	c.BytesCopied += int64(bytesRead)
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
