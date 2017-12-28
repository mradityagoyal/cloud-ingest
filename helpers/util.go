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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
)

// RetryWithExponentialBackoff tries the given function until it succeeds,
// using exponential back off when errors occur. When a failure occurs,
// an error message that includes functionName is printed and the sleepTime
// is increased (though the sleep time will never exceed maxSleepTime). After
// maxFails failures in a row, the method returns with an error. If maxFails
// is less than or equal to 0, the function is retried indefinitely until
// success. Both sleepTime and maxSleepTime must be greater than 0, else
// an error is returned.
func RetryWithExponentialBackoff(sleepTime time.Duration,
	maxSleepTime time.Duration, maxFails int, functionName string,
	function func() error) error {
	// TODO(b/65115935): Add jitter to the sleep time

	if sleepTime <= 0 {
		return fmt.Errorf("RetryWithExponentialBackoff: sleepTime must be greater "+
			"than 0. Current value: %v", sleepTime)
	}
	if maxSleepTime <= 0 {
		return fmt.Errorf("RetryWithExponentialBackoff: maxSleepTime must be "+
			"greater than 0. Current value: %v", maxSleepTime)
	}

	failures := 0
	for {
		err := function()
		if err == nil {
			return nil
		}
		failures++
		glog.Warningf("Error occurred in %s: %v.", functionName, err)

		if maxFails > 0 && failures >= maxFails {
			// Has failed maxFails times in a row, return with error
			return fmt.Errorf("aborting calls to %s after %d failures in a row.",
				functionName, maxFails)
		}

		glog.Infof("Retrying in %v.", sleepTime)
		time.Sleep(sleepTime)

		if sleepTime > maxSleepTime/2 {
			// sleepTime * 2 will be greater than maxSleepTime, just use maxSleepTime
			sleepTime = maxSleepTime
		} else {
			sleepTime *= 2
		}
	}
	return nil
}

// GetRelPathOsAgnostic wraps the path/filepath package's Rel function, and does some
// basic sanitization for Windows and NFS paths.
//
// Go's filepath.Rel function internally relies on the os package, which sets a different
// 'Separator' constant depending on which OS it is compiled for. Since we're compiling
// for *nix, none of the filepath or path package functions will work correctly on
// Windows or NFS directories (i.e., it doesn't recognize backslash as a separator).
func GetRelPathOsAgnostic(root, file string) string {
	rootBslashes := strings.Count(root, "\\")
	rootFslashes := strings.Count(root, "/")
	var rootHasDrivePrefix bool
	if len(root) >= 2 && unicode.IsLetter([]rune(root)[0]) && root[1] == ':' {
		rootHasDrivePrefix = true
	}

	fileBslashes := strings.Count(file, "\\")
	fileFslashes := strings.Count(file, "/")
	var fileHasDrivePrefix bool
	if len(file) >= 2 && unicode.IsLetter([]rune(file)[0]) && file[1] == ':' {
		fileHasDrivePrefix = true
	}

	// We have a *nix path, we can directly use Go's filepath.Rel(...) function.
	if rootBslashes == 0 && fileBslashes == 0 && !rootHasDrivePrefix && !fileHasDrivePrefix && (rootFslashes >= 0 || fileFslashes >= 0) {
		relPath, _ := filepath.Rel(root, file)
		return relPath
	}

	// If we have a Windows or NFS path (say, D:\dir\file0, or \\dir\file1) then
	// sanitize the paths so filepath.Rel(...) can operate as if they're *nix paths.
	if rootHasDrivePrefix {
		root = root[2:] // Strip the drive prefix.
	}
	if rootBslashes >= 0 && rootFslashes == 0 {
		root = strings.Replace(root, "\\", "/", -1)
	}

	if fileHasDrivePrefix {
		file = file[2:] // Strip the drive prefix.
	}
	if fileBslashes >= 0 && fileFslashes == 0 {
		file = strings.Replace(file, "\\", "/", -1)
	}

	relPath, _ := filepath.Rel(root, file)
	return relPath
}

// ToInt64 takes an arbitrary value known to be an integer, and
// converts it to an int64.
func ToInt64(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	default:
		return 0, fmt.Errorf("invalid int64 value %v (%T)", val, val)
	}
}

func GetGCSGenerationNumCondition(generationNum int64) storage.Conditions {
	if generationNum == 0 {
		return storage.Conditions{DoesNotExist: true}
	}
	return storage.Conditions{GenerationMatch: generationNum}
}
