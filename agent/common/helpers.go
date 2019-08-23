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
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	mountDirectory  = "/transfer_root"
	enableDirPrefix = flag.Bool("enable-directory-prefix", false, "This flag is used when agent runs inside container to prepend transfer_root as to the src directory.")
)

// SetMountDirectory is used in test to set the mount directory.
func SetMountDirectory(dir string) {
	mountDirectory = dir
}

// TaskFailureMsg removes the mount directory from the error message if agent is enabled to use mount directory.
func TaskFailureMsg(err error) string {
	if err == nil {
		return ""
	}

	failureMsg := fmt.Sprint(err)
	if *enableDirPrefix {
		failureMsg = strings.ReplaceAll(failureMsg, " " + mountDirectory + "/", " /")
	}
	return failureMsg
}

// OSPath prepends mount directory to the path if agent is enabled to use mount directory. Otherwise, returns the path without any changes.
func OSPath(path string) string {
	if *enableDirPrefix {
		return filepath.Join(mountDirectory, path)
	}
	return path
}

// LogDir removes the mount directory from the log directory if agent is enabled to use mount directory.
func LogDir(dir string) string {
	if *enableDirPrefix {
		dir = strings.Replace(dir, mountDirectory, "", 1)
	}
	return dir
}
