/*
Copyright 2019 Google Inc. All Rights Reserved.
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

package common

import (
	"errors"
	"flag"
	"testing"
)

func TestTaskFailureMsg(t *testing.T) {
	flag.Lookup("enable-directory-prefix").Value.Set("true")
	defer flag.Lookup("enable-directory-prefix").Value.Set("false")

	tests := []struct {
		desc		string
		err		error
		expectedMessage string
	}{
		{
			desc: "test mount directory appears once",
			err: errors.New("Open /transfer_root/text.txt failed, permission denied."),
			expectedMessage: "Open /text.txt failed, permission denied.",
		},
		{
			desc: "test mount directory appears multiple times in different paths",
			err: errors.New("Open /transfer_root/text1.txt and /transfer_root/text2.txt failed, permission denied."),
			expectedMessage: "Open /text1.txt and /text2.txt failed, permission denied.",
		},
		{
			desc: "test mount directory appears multiple times in one path, multiple paths in the same error",
			err: errors.New("Open /transfer_root/transfer_root/transfer_root.txt and /transfer_root/transfer_root/transfer_root/transfer_root.txt failed, permission denied."),
			expectedMessage: "Open /transfer_root/transfer_root.txt and /transfer_root/transfer_root/transfer_root.txt failed, permission denied.",
		},
	}

	for _, tc := range tests {
		gotMessage := TaskFailureMsg(tc.err)
		if gotMessage != tc.expectedMessage {
			t.Errorf("TaskFailureMsg(%q) failed, got: %s, want: %s", tc.desc, gotMessage, tc.expectedMessage)
		}
	}
}

func TestLogDir(t *testing.T) {
	flag.Lookup("enable-directory-prefix").Value.Set("true")
	defer flag.Lookup("enable-directory-prefix").Value.Set("false")

	tests := []struct {
		desc string
		logDir string
		wantLogDir string
	}{
		{
			desc: "mount directory appears once in log dir",
			logDir: "/transfer_root/tmp",
			wantLogDir: "/tmp",
		},
		{
			desc: "mount directory appears multiple times in log dir",
			logDir: "/transfer_root/transfer_root/tmp",
			wantLogDir: "/transfer_root/tmp",
		},
	}

	for _, tc := range tests {
		got := LogDir(tc.logDir)
		if got != tc.wantLogDir {
			t.Errorf("LogDir failed(%q), got: %s, want: %s", tc.desc, got, tc.wantLogDir)
		}
	}
}
