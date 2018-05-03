package agent

import (
	"errors"
	"fmt"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func checkCopyTaskSpec(c *taskpb.CopySpec) (resumedCopy bool, err error) {
	if c.SrcFile == "" {
		return false, errors.New("empty SrcFile")
	} else if c.DstBucket == "" {
		return false, errors.New("empty DstBucket")
	} else if c.DstObject == "" {
		return false, errors.New("empty DstObject")
	} else if c.ExpectedGenerationNum < 0 {
		return false, fmt.Errorf("invalid ExpectedGen'Num: %v", c.ExpectedGenerationNum)
	}

	if c.FileBytes != 0 || c.FileMTime != 0 || c.BytesCopied != 0 || c.Crc32C != 0 || c.ResumableUploadId != "" {
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
