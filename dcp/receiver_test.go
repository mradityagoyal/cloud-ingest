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
	"reflect"
	"testing"

	"github.com/golang/groupcache/lru"
)

func TestGetJobSpecFromCache(t *testing.T) {
	log.SetOutput(ioutil.Discard) // Temporarily suppress logging.
	defer log.SetOutput(os.Stdout)

	initialJobSpec := &JobSpec{
		OnpremSrcDirectory: "dummy-src",
		GCSBucket:          "dummy-bucket",
	}
	store := &FakeStore{
		jobSpec: initialJobSpec,
	}
	r := MessageReceiver{
		Store: store,
	}
	r.jobSpecsCache.c = lru.New(1)

	// The first call to getJobSepc puts the job spec in the cache.
	jobConfigRRStruct := JobConfigRRStruct{"dummy-project", "dummy-config"}
	jobSpec, _ := r.getJobSpec(jobConfigRRStruct)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}

	// Change the stored job from the cached spec and make sure the
	// MessageReceiver gets it from cache.
	store.jobSpec = &JobSpec{
		OnpremSrcDirectory: "modified-dummy-src",
		GCSBucket:          "modified-dummy-bucket",
	}

	// The second call should get the job spec from the cache.
	jobSpec, _ = r.getJobSpec(jobConfigRRStruct)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}
}

func TestGetJobSpecThatRemovedFromCache(t *testing.T) {
	log.SetOutput(ioutil.Discard) // Temporarily suppress logging.
	defer log.SetOutput(os.Stdout)

	initialJobSpec := &JobSpec{
		OnpremSrcDirectory: "dummy-src",
		GCSBucket:          "dummy-bucket",
	}
	store := &FakeStore{
		jobSpec: initialJobSpec,
	}
	r := MessageReceiver{
		Store: store,
	}
	r.jobSpecsCache.c = lru.New(1)

	// The first call to getJobSepc puts the job spec in the cache.
	jobConfigRRStruct1 := JobConfigRRStruct{"dummy-project", "dummy-config-1"}
	jobSpec, _ := r.getJobSpec(jobConfigRRStruct1)
	if !reflect.DeepEqual(jobSpec, initialJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}

	// Add another item in the cache so the first one got removed.
	jobConfigRRStruct2 := JobConfigRRStruct{"dummy-project", "dummy-config-2"}
	r.getJobSpec(jobConfigRRStruct2)

	// Change the stored job from the cached spec and make sure the
	// MessageReceiver gets it from cache.
	storedJobSpec := &JobSpec{
		OnpremSrcDirectory: "modified-dummy-src",
		GCSBucket:          "modified-dummy-bucket",
	}
	store.jobSpec = storedJobSpec

	// Reading the removed job spec should come from the store, not the cache.
	jobSpec, _ = r.getJobSpec(jobConfigRRStruct1)
	if !reflect.DeepEqual(jobSpec, storedJobSpec) {
		t.Errorf("expected getting job spec %v, but got %v", initialJobSpec, jobSpec)
	}
}
