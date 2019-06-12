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

package gcloud

import (
	"context"
	"io"
	"log"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// Pass-through wrapper for Google Cloud Storage client.
type GCS interface {
	CreateBucket(ctx context.Context, projectId, bucketName string, attrs *storage.BucketAttrs) error
	DeleteBucket(ctx context.Context, bucketName string) error
	DeleteObject(ctx context.Context, bucketName, objectName string, genNumber int64) error
	GetAttrs(ctx context.Context, bucketName, objectName string) (*storage.ObjectAttrs, error)
	ListObjects(ctx context.Context, bucketName string, query *storage.Query) ObjectIterator
	NewRangeReader(ctx context.Context, bucketName, objectName string, offset, length int64) (io.ReadCloser, error)
	NewWriter(ctx context.Context, bucketName, objectName string) WriteCloserWithError
	NewWriterWithCondition(ctx context.Context, bucketName, objectName string,
		cond storage.Conditions) WriteCloserWithError
}

type WriteCloserWithError interface {
	io.WriteCloser
	CloseWithError(err error) error
	Attrs() *storage.ObjectAttrs
}

// Interface to abstract out the ObjectIterator type, which contains
// hidden implementation details.
type ObjectIterator interface {
	Next() (*storage.ObjectAttrs, error)
}

type GCSClient struct {
	client *storage.Client
}

func NewGCSClient(client *storage.Client) *GCSClient {
	return &GCSClient{client}
}

// Pass-through method implementations.

func (gcs *GCSClient) CreateBucket(ctx context.Context, projectId, bucketName string, attrs *storage.BucketAttrs) error {
	return gcs.client.Bucket(bucketName).Create(ctx, projectId, attrs)
}

func (gcs *GCSClient) DeleteBucket(ctx context.Context, bucketName string) error {
	return gcs.client.Bucket(bucketName).Delete(ctx)
}

func (gcs *GCSClient) DeleteObject(ctx context.Context, bucketName, objectName string, genNumber int64) error {
	// Object generation number should only be used as a pre-condition to delete an object. This ensures that the right version of the
	// object is deleted and does not prohibit creating an archive. If generation number is passed in as an attribute, the object and
	// its corresponding archive will be deleted, leaving the customer with no recourse.
	return gcs.client.Bucket(bucketName).Object(objectName).If(storage.Conditions{GenerationMatch: genNumber}).Delete(ctx)
}

func (gcs *GCSClient) GetAttrs(ctx context.Context, bucketName, objectName string) (*storage.ObjectAttrs, error) {
	return gcs.client.Bucket(bucketName).Object(objectName).Attrs(ctx)
}

func (gcs *GCSClient) ListObjects(ctx context.Context, bucketName string, query *storage.Query) ObjectIterator {
	return gcs.client.Bucket(bucketName).Objects(ctx, query)
}

func (gcs *GCSClient) NewRangeReader(ctx context.Context, bucketName, objectName string, offset, length int64) (io.ReadCloser, error) {
	return gcs.client.Bucket(bucketName).Object(objectName).NewRangeReader(ctx, offset, length)
}

func (gcs *GCSClient) NewWriter(ctx context.Context,
	bucketName, objectName string) WriteCloserWithError {

	return gcs.client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
}

func (gcs *GCSClient) NewWriterWithCondition(ctx context.Context,
	bucketName, objectName string, cond storage.Conditions) WriteCloserWithError {

	return gcs.client.Bucket(bucketName).Object(objectName).If(cond).NewWriter(ctx)
}

// NewObjectIterator returns an in-memory instance of ObjectIterator. Prefer this approach
// when mocking ListObjects, over setting up a mock of ObjectIterator.
//
// Pass in either *storage.ObjectAttrs, or error, and it will do the right thing.
func NewObjectIterator(items ...interface{}) ObjectIterator {
	return &fakeObjectIterator{items: items}
}

type fakeObjectIterator struct {
	items []interface{}
	pos   int
}

func (iter *fakeObjectIterator) Next() (*storage.ObjectAttrs, error) {
	if iter.pos < len(iter.items) {
		item := iter.items[iter.pos]
		iter.pos++

		switch v := item.(type) {
		case *storage.ObjectAttrs:
			return v, nil
		case error:
			return nil, v
		default:
			log.Fatalf("item %v is neither *storage.ObjectAttr nor error", v)
		}
	}

	return nil, iterator.Done
}
