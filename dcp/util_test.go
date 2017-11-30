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
	"testing"
)

func TestGetRelPathOsAgnostic(t *testing.T) {
	var tests = []struct {
		root            string
		file            string
		expectedRelPath string
	}{
		// *nix examples.
		{"/dir/", "/dir/file0", "file0"},
		{"/dir", "/dir/file0", "file0"},
		{"/dir", "/dir/subdir/file0", "subdir/file0"},
		{"/dir/subdir", "/dir/subdir/file0", "file0"},
		{"/dir/subdir/", "/dir/subdir/file0", "file0"},

		// Windows examples.
		{"C:\\dir\\", "C:\\dir\\file0", "file0"},
		{"C:\\dir", "C:\\dir\\file0", "file0"},
		{"C:\\dir\\", "C:\\dir\\subdir\\file0", "subdir/file0"},
		{"C:\\dir\\subdir", "C:\\dir\\subdir\\file0", "file0"},
		{"C:\\dir\\subdir\\", "C:\\dir\\subdir\\file0", "file0"},

		// NFS examples.
		{"\\\\dir\\", "\\\\dir\\file0", "file0"},
		{"\\\\dir", "\\\\dir\\file0", "file0"},
		{"\\\\dir\\", "\\\\dir\\subdir\\file0", "subdir/file0"},
		{"\\\\dir\\subdir\\", "\\\\dir\\subdir\\file0", "file0"},
		{"\\\\dir\\subdir", "\\\\dir\\subdir\\file0", "file0"},
	}
	for _, tc := range tests {
		relPath := GetRelPathOsAgnostic(tc.root, tc.file)
		if relPath != tc.expectedRelPath {
			t.Errorf("Expected relPath %s for root %s and file %s, instead saw: %s",
				tc.expectedRelPath, tc.root, tc.file, relPath)
		}
	}
}
