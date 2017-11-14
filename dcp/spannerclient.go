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
	"cloud.google.com/go/spanner"
	"context"
	"time"
)

// Spanner wraps a spanner client.
type Spanner interface {
	Single() ReadOnlyTransaction
	ReadWriteTransaction(ctx context.Context, f func(context.Context, ReadWriteTransaction) error) (time.Time, error)
}

type ReadOnlyTransaction interface {
	Query(ctx context.Context, statement spanner.Statement) RowIterator
	Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) RowIterator
	ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
}

type ReadWriteTransaction interface {
	ReadOnlyTransaction
	BufferWrite(ms []*spanner.Mutation) error
}

type RowIterator interface {
	Do(f func(r *spanner.Row) error) error
	Next() (*spanner.Row, error)
	Stop()
}

// Default Implementations.

type SpannerClient struct {
	client *spanner.Client
}

func NewSpannerClient(client *spanner.Client) *SpannerClient {
	return &SpannerClient{client}
}
func (s *SpannerClient) Single() ReadOnlyTransaction {
	return &SpannerReadOnlyTransaction{s.client.Single()}
}

func (s *SpannerClient) ReadWriteTransaction(ctx context.Context, f func(context.Context, ReadWriteTransaction) error) (time.Time, error) {
	return s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return f(ctx, &SpannerReadWriteTransaction{txn})
	})
}

type SpannerReadOnlyTransaction struct {
	txn *spanner.ReadOnlyTransaction
}

func (txn *SpannerReadOnlyTransaction) Query(ctx context.Context, statement spanner.Statement) RowIterator {
	return &SpannerRowIterator{txn.txn.Query(ctx, statement)}
}

func (txn *SpannerReadOnlyTransaction) Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) RowIterator {
	return &SpannerRowIterator{txn.txn.Read(ctx, table, keys, columns)}
}

func (txn *SpannerReadOnlyTransaction) ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error) {
	return txn.txn.ReadRow(ctx, table, key, columns)
}

type SpannerReadWriteTransaction struct {
	txn *spanner.ReadWriteTransaction
}

func (txn *SpannerReadWriteTransaction) BufferWrite(ms []*spanner.Mutation) error {
	return txn.txn.BufferWrite(ms)
}

func (txn *SpannerReadWriteTransaction) Query(ctx context.Context, statement spanner.Statement) RowIterator {
	return &SpannerRowIterator{txn.txn.Query(ctx, statement)}
}

func (txn *SpannerReadWriteTransaction) Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) RowIterator {
	return &SpannerRowIterator{txn.txn.Read(ctx, table, keys, columns)}
}

func (txn *SpannerReadWriteTransaction) ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error) {
	return txn.txn.ReadRow(ctx, table, key, columns)
}

type SpannerRowIterator struct {
	iter *spanner.RowIterator
}

func (iter *SpannerRowIterator) Do(f func(r *spanner.Row) error) error {
	return iter.iter.Do(f)
}

func (iter *SpannerRowIterator) Next() (*spanner.Row, error) {
	return iter.iter.Next()
}

func (iter *SpannerRowIterator) Stop() {
	iter.iter.Stop()
}
