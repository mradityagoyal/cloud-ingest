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
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestHandleMessage(t *testing.T) {
	handler := LoadBQProgressMessageHandler{}
	taskCompletionMessage := &TaskCompletionMessage{
		FullTaskId: "J:R:T",
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{},
		LogEntry:   map[string]interface{}{},
	}
	taskUpdate, err := handler.HandleMessage(nil /* jobSpec */, taskCompletionMessage)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if taskUpdate.NewTasks != nil && len(taskUpdate.NewTasks) != 0 {
		t.Errorf("new tasks should be an empty array, new tasks: %v", taskUpdate.NewTasks)
	}
}

func TestLoadBQInvalidCompletionMsg(t *testing.T) {
	handler := LoadBQProgressMessageHandler{}
	taskCompletionMessage := &TaskCompletionMessage{
		FullTaskId: "garbage",
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{},
		LogEntry:   map[string]interface{}{},
	}
	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil /* jobSpec */, taskCompletionMessage)
	defer log.SetOutput(os.Stdout) // Reenable logging.

	if err == nil {
		t.Errorf("error is nil, expected error: %v.", errInvalidCompletionMessage)
	}
}
