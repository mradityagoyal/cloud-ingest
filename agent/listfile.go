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

// TODO(b/79217625): This code needs to be kept in sync with the DCP. Or at the very
// least taken into consideration for whatever versioning mechanism we develop. The
// same is true for listfile_test.go.

package agent

import (
	"fmt"
	"strings"
)

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
	if l.IsDir {
		typeField = "d"
	}
	return fmt.Sprintf("%v,%v", typeField, l.FilePath)
}
