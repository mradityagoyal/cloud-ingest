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

package dcp

import (
	"bufio"
	"context"
	"io"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
)

// ListResultReader is the interface that reads the listing task results from a
// GCS object.
type ListingResultReader interface {
	// ReadLines reads the listing task result from the GCS object identified by
	// bucket and object. It returns a slice of line entries (string) with maxLines or
	// fewer entries, and the new byte offset. When there are no more lines to be read
	// the error will io.EOF.
	ReadLines(ctx context.Context, bucket, object string, offset, maxLines int64) ([]string, int64, error)
}

type GCSListingResultReader struct {
	gcs gcloud.GCS
}

func NewGCSListingResultReader(gcs gcloud.GCS) *GCSListingResultReader {
	return &GCSListingResultReader{gcs}
}

func (r *GCSListingResultReader) ReadLines(ctx context.Context, bucket, object string, offset, maxLines int64) ([]string, int64, error) {
	sr, err := r.gcs.NewRangeReader(ctx, bucket, object, offset, -1)
	if err != nil {
		return nil, offset, err
	}
	defer sr.Close()

	discardedFirstEntry := false

	lines := make([]string, 0, maxLines)
	scanner := bufio.NewScanner(sr)
	for scanner.Scan() {
		line := scanner.Text()

		if offset == 0 && !discardedFirstEntry {
			// TODO(b/70793941): Remove the unused first line from the GCS listing file.
			discardedFirstEntry = true
			offset += int64(len(line)) + 1 // Extra byte for the stripped \n.
			continue
		}

		offset += int64(len(line)) + 1 // Extra byte for the stripped \n.

		lines = append(lines, scanner.Text())
		if int64(len(lines)) >= maxLines {
			// Check if this is the last line of the GCS listing file.
			if !scanner.Scan() {
				if err := scanner.Err(); err == nil {
					return lines, offset, io.EOF
				}
			}
			return lines, offset, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}
	return lines, offset, io.EOF
}
