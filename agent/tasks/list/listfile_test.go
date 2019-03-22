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

package list

import (
	"strings"
	"testing"
)

func TestListFileEntryParseAndStringSuccess(t *testing.T) {
	var tests = []struct {
		entry ListFileEntry
		line  string
	}{
		{ListFileEntry{true, "some path"}, "d,some path"},
		{ListFileEntry{true, "a/b/c/d"}, "d,a/b/c/d"},
		{ListFileEntry{true, "/a/b/c/d"}, "d,/a/b/c/d"},
		{ListFileEntry{true, "//a/b/c/d"}, "d,//a/b/c/d"},
		{ListFileEntry{true, "a\\b\\c\\d"}, "d,a\\b\\c\\d"},
		{ListFileEntry{true, "c:\\a\\b\\c\\d"}, "d,c:\\a\\b\\c\\d"},
		{ListFileEntry{true, "a,b"}, "d,a,b"},
		{ListFileEntry{false, "some path"}, "f,some path"},
		{ListFileEntry{false, "a/b/c/d"}, "f,a/b/c/d"},
		{ListFileEntry{false, "/a/b/c/d"}, "f,/a/b/c/d"},
		{ListFileEntry{false, "//a/b/c/d"}, "f,//a/b/c/d"},
		{ListFileEntry{false, "a\\b\\c\\d"}, "f,a\\b\\c\\d"},
		{ListFileEntry{false, "c:\\a\\b\\c\\d"}, "f,c:\\a\\b\\c\\d"},
		{ListFileEntry{false, "a,b"}, "f,a,b"},
	}
	for _, tc := range tests {
		parsedEntry, err := ParseListFileLine(tc.line)
		if err != nil {
			t.Errorf("Error parsing line %v, err: %v", tc.line, err)
		}
		if *parsedEntry != tc.entry {
			t.Errorf("Expected parsed %v, actual: %v", tc.entry, *parsedEntry)
		}
		if s := tc.entry.String(); s != tc.line {
			t.Errorf("Expected entry string %v, actual: %v", tc.line, s)
		}
	}
}

func TestListFileEntryParseFailure(t *testing.T) {
	// Parse fails without the correct number of fields.
	expectedErr := "expected 2 fields"
	if _, err := ParseListFileLine("some path with no delimiter"); err == nil {
		t.Errorf("error is nil, expected error: %v...", expectedErr)
	} else if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %s, found: %s.", expectedErr, err.Error())
	}

	// Parse fails with a bogus type field.
	expectedErr = "expected 'd' or 'f'"
	if _, err := ParseListFileLine("b,bogus type field"); err == nil {
		t.Errorf("error is nil, expected error: %v...", expectedErr)
	} else if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %s, found: %s.", expectedErr, err.Error())
	}
}
