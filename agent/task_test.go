package agent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Testing helper function.
func deepEqualCompare(msgPrefix string, want, got interface{}, t *testing.T) {
	if !reflect.DeepEqual(want, got) {
		t.Errorf("%s: Wanted %+v; got %+v", msgPrefix, want, got)
	}
}

func TestCopyTaskSpecFromTaskParams(t *testing.T) {
	var tests = []struct {
		// Mandatory fields.
		srcFile interface{}
		bucket  interface{}
		object  interface{}
		genNum  interface{}

		// Optional bandwidth field.
		bandwidth interface{}

		// Optional resumable copy fields.
		fileBytes         interface{}
		fileMtime         interface{}
		bytesCopied       interface{}
		crc32c            interface{}
		bytesToCopy       interface{}
		resumableUploadId interface{}
	}{
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, nil, nil, nil, nil, nil, nil},
		{"srcfile", "bucket", "object", int(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", json.Number("1"), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{nil, "bucket", "object", int64(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", nil, "object", int64(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", nil, int64(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", nil, 50, 2, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, nil, 3, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, 2, nil, 4, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, nil, 5, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, 4, nil, 6, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, 4, 5, nil, "resumableUploadId"},
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, 4, 5, 6, nil},
		{"srcfile", "bucket", "object", int64(1), 50, 2, 3, 4, 5, 6, "resumableUploadId"},
	}

	for _, tc := range tests {
		params := make(map[string]interface{})
		if tc.srcFile != nil {
			params["src_file"] = tc.srcFile
		}
		if tc.bucket != nil {
			params["dst_bucket"] = tc.bucket
		}
		if tc.object != nil {
			params["dst_object"] = tc.object
		}
		if tc.genNum != nil {
			params["expected_generation_num"] = tc.genNum
		}
		if tc.bandwidth != nil {
			params["bandwidth"] = tc.bandwidth
		}
		if tc.fileBytes != nil {
			params["file_bytes"] = tc.fileBytes
		}
		if tc.fileMtime != nil {
			params["file_mtime"] = tc.fileMtime
		}
		if tc.bytesCopied != nil {
			params["bytes_copied"] = tc.bytesCopied
		}
		if tc.crc32c != nil {
			params["crc32c"] = tc.crc32c
		}
		if tc.bytesToCopy != nil {
			params["bytes_to_copy"] = tc.bytesToCopy
		}
		if tc.resumableUploadId != nil {
			params["resumable_upload_id"] = tc.resumableUploadId
		}

		result, err := copyTaskSpecFromTaskParams(params)

		if tc.srcFile != nil && tc.bucket != nil && tc.object != nil && tc.genNum != nil &&
			tc.fileBytes != nil && tc.fileMtime != nil && tc.bytesCopied != nil &&
			tc.crc32c != nil && tc.bytesToCopy != nil && tc.resumableUploadId != nil {
			// All values populated, should be working result (always same values).
			expected := &copyTaskSpec{
				SrcFile:               "srcfile",
				DstBucket:             "bucket",
				DstObject:             "object",
				ExpectedGenerationNum: 1,
				Bandwidth:             50,
				FileBytes:             2,
				FileMtime:             3,
				BytesCopied:           4,
				CRC32C:                5,
				BytesToCopy:           6,
				ResumableUploadId:     "resumableUploadId",
			}
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			deepEqualCompare("copyTaskSpec construction from map", expected, result, t)
		} else if tc.srcFile != nil && tc.bucket != nil && tc.object != nil && tc.genNum != nil &&
			tc.fileBytes == nil && tc.fileMtime == nil && tc.bytesCopied == nil &&
			tc.crc32c == nil && tc.bytesToCopy == nil && tc.resumableUploadId == nil {
			var expectedBandwidth int64
			if tc.bandwidth != nil {
				expectedBandwidth = 50
			}
			// Mandatory values populated, no resumable values present.
			expected := &copyTaskSpec{
				SrcFile:               "srcfile",
				DstBucket:             "bucket",
				DstObject:             "object",
				ExpectedGenerationNum: 1,
				Bandwidth:             expectedBandwidth,
			}
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			deepEqualCompare("copyTaskSpec construction from map", expected, result, t)
		} else {
			// Any missing parameter should result in error.
			if err == nil {
				t.Errorf("wanted: missing params, but got nil error, tc %v", tc)
			} else if !strings.Contains(err.Error(), "missing params") {
				t.Errorf("wanted: missing params, but got: %v", err)
			}
		}
	}
}

func TestCheckCopyTaskSpec(t *testing.T) {
	type w struct {
		resumedCopy bool
		err         string
	}
	var tests = []struct {
		cts  copyTaskSpec
		want w
	}{
		// Non-resumed copy.
		{copyTaskSpec{"f", "b", "o", 0, 0, 0, 0, 0, 0, 0, ""}, w{false, ""}},
		{copyTaskSpec{"", "b", "o", 0, 0, 0, 0, 0, 0, 0, ""}, w{false, "empty SrcFile"}},
		{copyTaskSpec{"f", "", "o", 0, 0, 0, 0, 0, 0, 0, ""}, w{false, "empty DstBucket"}},
		{copyTaskSpec{"f", "b", "", 0, 0, 0, 0, 0, 0, 0, ""}, w{false, "empty DstObject"}},
		{copyTaskSpec{"f", "b", "o", -1, 0, 0, 0, 0, 0, 0, ""}, w{false, "invalid ExpectedGen"}},

		// Resumed copy.
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 1, 10, 99, 10, "ruID"}, w{true, ""}},
		{copyTaskSpec{"f", "b", "o", 0, 0, -1, 1, 10, 99, 10, "ruID"}, w{true, "but FileBytes"}},
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 0, 10, 99, 10, "ruID"}, w{true, ""}}, // mtime 0 ok.
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 1, -1, 99, 10, "ruID"}, w{true, "but BytesCopied"}},
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 1, 10, 0, 10, "ruID"}, w{true, ""}}, // CRC32C 0 ok.
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 1, 10, 99, 0, "ruID"}, w{true, ""}}, // b'ToCopy 0 ok.
		{copyTaskSpec{"f", "b", "o", 0, 0, 20, 1, 10, 99, 10, ""}, w{true, "empty ResumableUploadId"}},
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
