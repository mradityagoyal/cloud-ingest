package copy

import (
	"strings"
	"testing"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func tCopySpec(sf, db, do string, egn, bw, fb, fm, bc, btc int64, crc uint32, ruID string) *taskpb.CopySpec {
	return &taskpb.CopySpec{
		SrcFile:               sf,
		DstBucket:             db,
		DstObject:             do,
		ExpectedGenerationNum: egn,
		Bandwidth:             bw,
		FileBytes:             fb,
		FileMTime:             fm,
		BytesCopied:           bc,
		Crc32C:                crc,
		BytesToCopy:           btc,
		ResumableUploadId:     ruID,
	}
}

func TestCheckCopyTaskSpec(t *testing.T) {
	type w struct {
		resumedCopy bool
		err         string
	}
	var tests = []struct {
		cts  *taskpb.CopySpec
		want w
	}{
		// Non-resumed copy.
		{tCopySpec("f", "b", "o", 0, 0, 0, 0, 0, 0, 0, ""), w{false, ""}},
		{tCopySpec("", "b", "o", 0, 0, 0, 0, 0, 0, 0, ""), w{false, "empty SrcFile"}},
		{tCopySpec("f", "", "o", 0, 0, 0, 0, 0, 0, 0, ""), w{false, "empty DstBucket"}},
		{tCopySpec("f", "b", "", 0, 0, 0, 0, 0, 0, 0, ""), w{false, "empty DstObject"}},
		{tCopySpec("f", "b", "o", -1, 0, 0, 0, 0, 0, 0, ""), w{false, "invalid ExpectedGen"}},

		// Resumed copy.
		{tCopySpec("f", "b", "o", 0, 0, 20, 1, 10, 99, 10, "ruID"), w{true, ""}},
		{tCopySpec("f", "b", "o", 0, 0, -1, 1, 10, 99, 10, "ruID"), w{true, "but FileBytes"}},
		{tCopySpec("f", "b", "o", 0, 0, 20, 0, 10, 99, 10, "ruID"), w{true, ""}}, // mtime 0 ok.
		{tCopySpec("f", "b", "o", 0, 0, 20, 1, -1, 99, 10, "ruID"), w{true, "but BytesCopied"}},
		{tCopySpec("f", "b", "o", 0, 0, 20, 1, 10, 0, 10, "ruID"), w{true, ""}}, // CRC32C 0 ok.
		{tCopySpec("f", "b", "o", 0, 0, 20, 1, 10, 99, 0, "ruID"), w{true, ""}}, // b'ToCopy 0 ok.
		{tCopySpec("f", "b", "o", 0, 0, 20, 1, 10, 99, 10, ""), w{true, "empty ResumableUploadId"}},
	}

	for _, tc := range tests {
		resumedCopy, err := checkCopyTaskSpec(tc.cts)
		if tc.want.resumedCopy != resumedCopy {
			t.Errorf("resumedCopy want %v, got %v, tc %v", tc.want.resumedCopy, resumedCopy, tc)
		}
		if tc.want.err == "" {
			if err != nil {
				t.Errorf("err want nil, got %v, tc %v", err, tc)
			}
		} else {
			if err == nil {
				t.Errorf("err want '%v', got nil, tc %v", tc.want.err, tc)
			} else if !strings.Contains(err.Error(), tc.want.err) {
				t.Errorf("err contains want '%v', got '%v', tc %v", tc.want.err, err, tc)
			}
		}
	}
}
