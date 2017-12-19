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
	"bytes"
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

// StringWriteCloser implements WriteCloser interface for faking storage.Writer.
type StringWriteCloser struct {
	buffer bytes.Buffer
	closed bool

	// attrs fakes the storage object attributes generated after write completion.
	attrs *storage.ObjectAttrs
}

func NewStringWriteCloser(attrs *storage.ObjectAttrs) *StringWriteCloser {
	return &StringWriteCloser{attrs: attrs}
}

func (m *StringWriteCloser) Write(p []byte) (int, error) {
	return m.buffer.Write(p)
}

func (m *StringWriteCloser) Close() error {
	m.closed = true
	return nil
}

func (m *StringWriteCloser) CloseWithError(err error) error {
	return nil
}

func (m *StringWriteCloser) Attrs() *storage.ObjectAttrs {
	if m.closed {
		return m.attrs
	}
	return nil
}

func (m *StringWriteCloser) WrittenString() string {
	if m.closed {
		return m.buffer.String()
	}
	return ""
}

func (m *StringWriteCloser) NumberLines() int64 {
	if m.closed {
		return CountLines(m.buffer.String())
	}
	return int64(0)
}

func CountLines(s string) int64 {
	return int64(len(strings.Split(strings.Trim(s, "\n"), "\n")))
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
