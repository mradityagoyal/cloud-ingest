/*
Copyright 2018 Google Inc. All Rights Reserved.
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

package agent

import (
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
)

func TestDirectoryInfoStoreAddSizeIncreases(t *testing.T) {
	testStr1 := "path/to/some/dir"
	testStr2 := "a"
	testStr3 := "another/path/to/a/dir/that/is/longer/than/the/others/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z"
	tests := []struct {
		desc    string
		dirInfo listpb.DirectoryInfo
	}{
		{
			desc:    "Add a DirInfo",
			dirInfo: listpb.DirectoryInfo{Path: testStr1},
		},
		{
			desc:    "Add a DirInfo with a short path",
			dirInfo: listpb.DirectoryInfo{Path: testStr2},
		},
		{
			desc:    "Add a DirInfo with a slightly longer path",
			dirInfo: listpb.DirectoryInfo{Path: testStr3},
		},
	}
	dirStore := NewDirectoryInfoStore()
	for _, tc := range tests {
		prevSize := dirStore.Size()
		if err := dirStore.Add(tc.dirInfo); err != nil {
			t.Fatalf("got error %v", err)
		}
		newSize := dirStore.Size()
		change := newSize - prevSize
		wantChange := len(tc.dirInfo.Path) + directoryInfoProtoOverhead
		if change != wantChange {
			t.Errorf("TestDirectoryInfoStore_Add(%q) wanted size change equal to %d, got %d", tc.desc, wantChange, change)
		}
	}
}

func TestDirectoryInfoStoreAddInvalidDirInfo(t *testing.T) {
	dirStore := NewDirectoryInfoStore()
	if err := dirStore.Add(listpb.DirectoryInfo{}); err == nil {
		t.Fatal("want error, got nil")
	}
	if length := len(dirStore.DirectoryInfos()); length > 0 {
		t.Fatalf("want len(dirStore.DirectoryInfos) == 0, got %d", length)
	}
}

func TestDirectoryInfoStoreDirectoryInfosSorted(t *testing.T) {
	dirInfos := []listpb.DirectoryInfo{{Path: "d"}, {Path: "b"}, {Path: "a"}, {Path: "c"}}
	dirStore := NewDirectoryInfoStore()
	for _, dirInfo := range dirInfos {
		if err := dirStore.Add(dirInfo); err != nil {
			t.Fatalf("got error %v", err)
		}
	}
	sort.Slice(dirInfos, func(i, j int) bool {
		return dirInfos[i].Path < dirInfos[j].Path
	})
	actualDirInfos := dirStore.DirectoryInfos()
	if !cmp.Equal(dirInfos, actualDirInfos) {
		t.Errorf("want sorted directory infos %v, got %v", dirInfos, actualDirInfos)
	}
}

func TestDirectoryInfoStoreRemoveFirst(t *testing.T) {
	dirStore := NewDirectoryInfoStore()
	dirInfos := []listpb.DirectoryInfo{{Path: "dir"}, {Path: "path/to/dir"}, {Path: "another/path/to/dir"}}
	for _, dirInfo := range dirInfos {
		if err := dirStore.Add(dirInfo); err != nil {
			t.Fatalf("got error %v", err)
		}
	}
	if !(dirStore.Size() > 0) {
		t.Fatalf("want dirStore.Size() > 0, got %d", dirStore.Size())
	}
	gotDirInfo := dirStore.RemoveFirst()
	if len(dirStore.directoryInfos) != len(dirInfos)-1 {
		t.Fatalf("want len(dirStore.directoryInfos) == %d, got %d", len(dirInfos)-1, len(dirStore.directoryInfos))
	}
	sort.Slice(dirInfos, func(i, j int) bool {
		return dirInfos[i].Path < dirInfos[j].Path
	})
	if !cmp.Equal(dirStore.DirectoryInfos(), dirInfos[1:]) {
		t.Errorf("want dirStore.directoryInfos == %v, got %v", dirInfos[1:], dirStore.DirectoryInfos())
	}
	if !proto.Equal(gotDirInfo, &dirInfos[0]) {
		t.Errorf("want %v, got %v", &dirInfos[0], gotDirInfo)
	}
}
