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
	"context"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

// DoPeriodicially executes the input function once immediately and then again
// each time that the ticker ticks.
// TODO(b/72226634): Refactor MessageReceiver and associated tests to use this.
func DoPeriodically(ctx context.Context, ticker helpers.Ticker, f func()) {
	// Ticker is used as the loop conditional to ensure the loop immediately runs once.
	for ; true; <-ticker.GetChannel() {
		select {
		case <-ctx.Done():
			glog.Warningf(
				"Context for %v was cancelled with context error: %v.", f, ctx.Err())
			return
		default:
			f()
		}
	}
}
