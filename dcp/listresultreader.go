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

	"golang.org/x/net/context"

	"cloud.google.com/go/storage"
)

// ListResultReader is the interface that reads the listing task results from a
// GCS object.
type ListingResultReader interface {
	// ReadListResult reads the listing task result from the GCS object identified
	// by bucketName and objectName. It returns a string channel that contains the
	// listed objects.
	ReadListResult(bucketName string, objectName string) (chan string, error)
}

type GCSListingResultReader struct {
	Client *storage.Client
}

// TODO(b/63014139): Add unit testing for this method.
func (r *GCSListingResultReader) ReadListResult(bucketName string, objectName string) (chan string, error) {
	c := make(chan string)

	sr, err := r.Client.Bucket(bucketName).Object(objectName).NewReader(context.Background())
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(c)
		defer sr.Close()

		scanner := bufio.NewScanner(sr)
		for scanner.Scan() {
			c <- scanner.Text()
		}
	}()
	return c, nil
}
