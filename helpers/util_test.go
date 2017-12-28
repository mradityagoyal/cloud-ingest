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

package helpers

import (
	"fmt"
	"testing"
	"time"
)

func TestRetryWithExponentialBackoffSuccessFirstTime(t *testing.T) {
	err := RetryWithExponentialBackoff(
		time.Hour, time.Hour, 1, "Test", func() error {
			return nil
		})
	if err != nil {
		t.Errorf("expected RetryWithExponentialBackoff to succeed, but found: %v", err)
	}
}

func TestRetryWithExponentialBackoffFail(t *testing.T) {
	err := RetryWithExponentialBackoff(
		time.Millisecond, 30*time.Millisecond, 3, "Test", func() error {
			return fmt.Errorf("error")
		})
	if err == nil {
		t.Errorf("expected RetryWithExponentialBackoff to fail")
	}
}

func TestRetryWithExponentialBackoffMaxTrials(t *testing.T) {
	count := 0
	err := RetryWithExponentialBackoff(
		time.Millisecond, 30*time.Millisecond, 3, "Test", func() error {
			count++
			if count < 4 {
				return fmt.Errorf("error")
			}
			return nil
		})
	// Expected to fail.
	if err == nil {
		t.Errorf("expected RetryWithExponentialBackoff to fail")
	}
	// Expected number of call to be 3.
	if count != 3 {
		t.Errorf("expected to be retried 3 times, but it retried %d time(s)", count)
	}
}

func TestRetryWithExponentialBackoffSucceedsOnLastTry(t *testing.T) {
	count := 0
	err := RetryWithExponentialBackoff(
		time.Millisecond, 30*time.Millisecond, 4, "Test", func() error {
			count++
			if count < 3 {
				return fmt.Errorf("error")
			}
			return nil
		})
	// Expected to succeed.
	if err != nil {
		t.Errorf("expected RetryWithExponentialBackoff to succeed but got err: %v", err)
	}
	// Expected number of call to be 3.
	if count != 3 {
		t.Errorf("expected to be retried 3 times, but it retried %d time(s)", count)
	}
}

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
