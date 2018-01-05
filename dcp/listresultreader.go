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
	"fmt"
	"io"
	"strings"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
)

// ListResultReader is the interface that reads the listing task results from a
// GCS object.
type ListingResultReader interface {
	// ReadEntries reads the listing task result from the GCS object identified by
	// bucket and object. It returns a slice of ListFileEntry's with maxEntries or
	// fewer entries, and the new byte offset. When there are no more entries to be
	// read the error will io.EOF.
	ReadEntries(ctx context.Context, bucket, object string, offset int64, maxEntries int) ([]ListFileEntry, int64, error)
}

type GCSListingResultReader struct {
	gcs gcloud.GCS
}

func NewGCSListingResultReader(gcs gcloud.GCS) *GCSListingResultReader {
	return &GCSListingResultReader{gcs}
}

func (r *GCSListingResultReader) ReadEntries(ctx context.Context, bucket, object string, offset int64, maxEntries int) ([]ListFileEntry, int64, error) {
	sr, err := r.gcs.NewRangeReader(ctx, bucket, object, offset, -1)
	if err != nil {
		return nil, offset, err
	}
	defer sr.Close()

	discardedFirstEntry := false

	listFileEntries := make([]ListFileEntry, 0, maxEntries)
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

		listFileEntry, err := ParseListFileLine(line)
		if err != nil {
			return nil, offset, fmt.Errorf("couldn't parse line %v, err: %v", line, err)
		}

		listFileEntries = append(listFileEntries, *listFileEntry)
		if len(listFileEntries) >= maxEntries {
			// Check if this is the last line of the GCS listing file.
			if !scanner.Scan() {
				if err := scanner.Err(); err == nil {
					return listFileEntries, offset, io.EOF
				}
			}
			return listFileEntries, offset, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}
	return listFileEntries, offset, io.EOF
}

type ListFileEntry struct {
	IsDir    bool
	FilePath string
}

func ParseListFileLine(line string) (*ListFileEntry, error) {
	var l ListFileEntry
	fields := strings.SplitN(line, ",", 2)
	if len(fields) != 2 {
		return nil, fmt.Errorf("expected 2 fields (got %v) for line %v", len(fields), line)
	}
	if fields[0] != "d" && fields[0] != "f" {
		return nil, fmt.Errorf("expected 'd' or 'f' type field (got %v), for line %v", fields[0], line)
	}
	l.IsDir = (fields[0] == "d")
	l.FilePath = fields[1]
	return &l, nil
}

func (l ListFileEntry) String() string {
	typeField := "f"
	if  l.IsDir {
		typeField = "d"
	}
	return fmt.Sprintf("%v,%v", typeField, l.FilePath)
}
