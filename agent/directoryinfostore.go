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

type DirectoryInfoStore struct {
	directoryInfos []listpb.DirectoryInfo
	size           int
}

func NewDirectoryInfoStore() *DirectoryInfoStore {
	return &DirectoryInfoStore{directoryInfos: make([]listpb.DirectoryInfo, 0), size: 0}
}

func (s *DirectoryInfoStore) Add(dirInfo listpb.DirectoryInfo) error {
	if len(dirInfo.Path) == 0 {
		return errors.New("DirectoryInfoStore.Add: dirInfo.Path cannot be an empty string")
	}
	index := sort.Search(len(s.directoryInfos), func(i int) bool {
		return s.directoryInfos[i].Path > dirInfo.Path
	})
	s.directoryInfos = append(s.directoryInfos, listpb.DirectoryInfo{})
	copy(s.directoryInfos[index+1:], s.directoryInfos[index:])
	s.directoryInfos[index] = dirInfo
	s.size += len(dirInfo.Path) + directoryInfoProtoOverhead
	return nil
}

func (s *DirectoryInfoStore) Size() int {
	return s.size
}

func (s *DirectoryInfoStore) DirectoryInfos() []listpb.DirectoryInfo {
	return s.directoryInfos
}

func (s *DirectoryInfoStore) RemoveFirst() *listpb.DirectoryInfo {
	if len(s.directoryInfos) == 0 {
		return nil
	}
	dirInfo := s.directoryInfos[0]
	s.directoryInfos = s.directoryInfos[1:]
	return &dirInfo
}
