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
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FakeRowIterator implements a limited fake of the Spanner client.
type FakeRowIterator struct {
	index int
	m     []spanner.Row
}

func NewFakeRowIterator(rows []spanner.Row) *FakeRowIterator {
	return &FakeRowIterator{0, rows}
}

func (f *FakeRowIterator) Do(fun func(r *spanner.Row) error) error {
	for _, row := range f.m {
		if err := fun(&row); err != nil {
			return err
		}
	}
	return nil
}

func (f *FakeRowIterator) Next() (*spanner.Row, error) {
	if f.index >= len(f.m) {
		return nil, iterator.Done
	}
	currentIndex := f.index
	f.index = f.index + 1 // Advance the iterator
	return &f.m[currentIndex], nil
}

func (f *FakeRowIterator) Stop() {
	f.index = len(f.m)
}
