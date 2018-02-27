package agent

import (
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

type copyTaskSpec struct {
	// Mandatory fields.
	SrcFile               string `json:"src_file"`
	DstBucket             string `json:"dst_bucket"`
	DstObject             string `json:"dst_object"`
	ExpectedGenerationNum int64  `json:"expected_generation_num"`

	// Optional field for bandwidth control.
	Bandwidth int64 // Bytes per second for this task.

	// Optional fields only for managing resumable copies.
	FileBytes         int64  `json:"file_bytes"`
	FileMtime         int64  `json:"file_mtime"`
	BytesCopied       int64  `json:"bytes_copied"`
	CRC32C            int64  `json:"crc32c"` // TODO(b/...): Make this a uint32.
	BytesToCopy       int64  `json:"bytes_to_copy"`
	ResumableUploadId string `json:"resumable_upload_id"`
}

type taskParams map[string]interface{}

// taskDoneMsg is the response the client sends to the DCP when a task is done.
type taskDoneMsg struct {
	TaskRRName       string                     `json:"task_rr_name"`
	Status           string                     `json:"status"`
	FailureType      proto.TaskFailureType_Type `json:"failure_reason"`
	FailureMessage   string                     `json:"failure_message"`
	LogEntry         dcp.LogEntry               `json:"log_entry"`
	TaskParams       taskParams                 `json:"task_params"`
	TaskParamUpdates taskParams                 `json:"task_param_updates"`
}

func copyTaskSpecFromTaskParams(params map[string]interface{}) (cts *copyTaskSpec, err error) {
	// Mandatory params.
	srcFile, ok1 := params["src_file"]
	dstBucket, ok2 := params["dst_bucket"]
	dstObject, ok3 := params["dst_object"]
	jsonNumGenNum, ok4 := params["expected_generation_num"]

	// Optional bandwidth param.
	jsonNumBandwidth, ok5 := params["bandwidth"]

	// Optional resumable params.
	jsonNumFileBytes, ok6 := params["file_bytes"]
	jsonNumFileMtime, ok7 := params["file_mtime"]
	jsonNumBytesCopied, ok8 := params["bytes_copied"]
	jsonNumCRC32C, ok9 := params["crc32c"]
	jsonNumBytesToCopy, ok10 := params["bytes_to_copy"]
	resumableUploadId, ok11 := params["resumable_upload_id"]

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil, fmt.Errorf("missing params in copyTaskSpec map: %v", params)
	}
	var genNum int64
	if genNum, err = helpers.ToInt64(jsonNumGenNum); err != nil {
		return nil, err
	}

	var bandwidth int64
	if ok5 {
		if bandwidth, err = helpers.ToInt64(jsonNumBandwidth); err != nil {
			return nil, err
		}
	}

	if ok6 || ok7 || ok8 || ok9 || ok10 || ok11 {
		// If any of these are set, they must all be set.
		if !ok6 || !ok7 || !ok8 || !ok9 || !ok10 || !ok11 {
			return nil, fmt.Errorf("missing params in copyTaskSpec map: %v", params)
		}
		var fileBytes, fileMtime, bytesCopied, crc32c, bytesToCopy int64
		if fileBytes, err = helpers.ToInt64(jsonNumFileBytes); err != nil {
			return nil, err
		}
		if fileMtime, err = helpers.ToInt64(jsonNumFileMtime); err != nil {
			return nil, err
		}
		if bytesCopied, err = helpers.ToInt64(jsonNumBytesCopied); err != nil {
			return nil, err
		}
		if crc32c, err = helpers.ToInt64(jsonNumCRC32C); err != nil {
			return nil, err
		}
		if bytesToCopy, err = helpers.ToInt64(jsonNumBytesToCopy); err != nil {
			return nil, err
		}
		return &copyTaskSpec{
			SrcFile:               srcFile.(string),
			DstBucket:             dstBucket.(string),
			DstObject:             dstObject.(string),
			ExpectedGenerationNum: genNum,
			Bandwidth:             bandwidth,
			FileBytes:             fileBytes,
			FileMtime:             fileMtime,
			BytesCopied:           bytesCopied,
			CRC32C:                crc32c,
			BytesToCopy:           bytesToCopy,
			ResumableUploadId:     resumableUploadId.(string),
		}, nil
	}

	return &copyTaskSpec{
		SrcFile:               srcFile.(string),
		DstBucket:             dstBucket.(string),
		DstObject:             dstObject.(string),
		ExpectedGenerationNum: genNum,
		Bandwidth:             bandwidth,
	}, nil
}

func checkCopyTaskSpec(c copyTaskSpec) (resumedCopy bool, err error) {
	if c.SrcFile == "" {
		return false, errors.New("empty SrcFile")
	} else if c.DstBucket == "" {
		return false, errors.New("empty DstBucket")
	} else if c.DstObject == "" {
		return false, errors.New("empty DstObject")
	} else if c.ExpectedGenerationNum < 0 {
		return false, fmt.Errorf("invalid ExpectedGen'Num: %v", c.ExpectedGenerationNum)
	}

	if c.FileBytes != 0 || c.FileMtime != 0 || c.BytesCopied != 0 || c.CRC32C != 0 || c.ResumableUploadId != "" {
		// A resumed copy must have appropriate values for all of these parameters.
		// Note1: we place no restrictions on what constitues a valid mtime.
		// Note2: A zero CRC32C is valid (just suspicious).
		// Note3: There's no need to check "BytesToCopy", all values have valid meanings.
		if c.FileBytes <= 0 {
			return true, fmt.Errorf("resumedCopy but FileBytes <= 0: %v", c.FileBytes)
		} else if c.BytesCopied <= 0 {
			return true, fmt.Errorf("resumedCopy but BytesCopied <= 0: %v", c.BytesCopied)
		} else if c.ResumableUploadId == "" {
			return true, errors.New("resumedCopy with empty ResumableUploadId")
		}
		return true, nil
	}

	return false, nil
}
