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
	"errors"
	"sort"

	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
)

const directoryInfoProtoOverhead = 48

// DirectoryInfoStore stores a sorted list of DirectoryInfos and keeps track of the approximate
// number of bytes used to store them.
type DirectoryInfoStore struct {
	directoryInfos []listpb.DirectoryInfo
	size           int
}

func NewDirectoryInfoStore() *DirectoryInfoStore {
	return &DirectoryInfoStore{directoryInfos: make([]listpb.DirectoryInfo, 0), size: 0}
}

// Add adds the given dirInfo to the DirectoryInfoStore.
// If the given dirInfo is already stored in the DirectoryInfoStore or an invalid dirInfo
// is passed (dirInfo.Path is not set), Add returns an error.
func (s *DirectoryInfoStore) Add(dirInfo listpb.DirectoryInfo) error {
	if len(dirInfo.Path) == 0 {
		return errors.New("DirectoryInfoStore.Add: dirInfo.Path cannot be an empty string")
	}
	index := sort.Search(len(s.directoryInfos), func(i int) bool {
		return s.directoryInfos[i].Path >= dirInfo.Path
	})
	if index != len(s.directoryInfos) && s.directoryInfos[index].Path == dirInfo.Path {
		// Directory is already in the store, don't add a duplicate
		return errors.New("DirectoryInfoStore.Add: given dirInfo is already present in store")
	}
	s.directoryInfos = append(s.directoryInfos, listpb.DirectoryInfo{})
	copy(s.directoryInfos[index+1:], s.directoryInfos[index:])
	s.directoryInfos[index] = dirInfo
	s.size += approximateSizeOfDirInfo(dirInfo)
	return nil
}

// Size returns an approximation of the bytes currently used by the DirectoryInfoStore.
func (s *DirectoryInfoStore) Size() int {
	return s.size
}

// DirectoryInfos returns a sorted list of the DirectoryInfos stored in the DirectoryInfoStore.
func (s *DirectoryInfoStore) DirectoryInfos() []listpb.DirectoryInfo {
	return s.directoryInfos
}

// RemoveFirst removes the first DirectoryInfo from the DirectoryInfoStore as determined by case
// sensitive alphabetical order. If there are no DirectoryInfos in the store, nil is returned.
func (s *DirectoryInfoStore) RemoveFirst() *listpb.DirectoryInfo {
	if len(s.directoryInfos) == 0 {
		return nil
	}
	dirInfo := s.directoryInfos[0]
	s.directoryInfos = s.directoryInfos[1:]
	s.size -= approximateSizeOfDirInfo(dirInfo)
	return &dirInfo
}

// Len returns the number of directories stored in the DirectoryInfoStore.
func (s *DirectoryInfoStore) Len() int {
	return len(s.directoryInfos)
}

func approximateSizeOfDirInfo(dirInfo listpb.DirectoryInfo) int {
	return len(dirInfo.Path) + directoryInfoProtoOverhead
}
