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
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"strings"

	"cloud.google.com/go/storage"
)

type stringReadCloser struct {
	reader io.Reader
	Closed bool
}

func (src *stringReadCloser) Read(p []byte) (int, error) {
	return src.reader.Read(p)
}

func (src *stringReadCloser) Close() error {
	src.Closed = true
	return nil
}

func NewStringReadCloser(s string) *stringReadCloser {
	return &stringReadCloser{strings.NewReader(s), false}
}

type LinesWriterCloser struct {
	Writer       io.Writer
	WrittenLines int64
}

func (m *LinesWriterCloser) Write(p []byte) (int, error) {
	m.WrittenLines++
	return m.Writer.Write(p)
}

func (m *LinesWriterCloser) Close() error {
	return nil
}

func (m *LinesWriterCloser) CloseWithError(err error) error {
	return nil
}

func (m *LinesWriterCloser) Attrs() *storage.ObjectAttrs {
	return nil
}

// AreEqualJson checkes if strings s1 and s2 are identical JSON represention
// for the same JSON objects.
// TODO(b/63159302): Add unit tests for util class.
func AreEqualJSON(s1, s2 string) bool {
	var o1 interface{}
	var o2 interface{}

	if err := json.Unmarshal([]byte(s1), &o1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(s2), &o2); err != nil {
		return false
	}

	return reflect.DeepEqual(o1, o2)
}

// CreateTmpFile creates a temp file in the os temp directory with a prefix and
// content string. This method will panic in case of failure writing the file.
func CreateTmpFile(filePrefix string, content string) string {
	tmpfile, err := ioutil.TempFile("", filePrefix)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	return tmpfile.Name()
}
