/*
Copyright 2018 Google Inc. All Rights Reserved.
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
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
)

// writeProtobuf writes the given protobuf message using the given writer.
func writeProtobuf(w io.Writer, pb proto.Message) error {
	pbStr := proto.MarshalTextString(pb)
	pbBytes := []byte(pbStr)
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(pbBytes)))
	_, err := w.Write(lenBytes)
	if err != nil {
		glog.Errorf("failed to write protobuf size %d with error %v", len(pbBytes), err)
		return err
	}
	_, err = w.Write(pbBytes)
	if err != nil {
		glog.Errorf("failed to write protobuf %v with error %v", pb, err)
		return err
	}
	return nil
}

// parseProtobuf parses the next protobuf from the reader and uses pb to hold the parsed data.
func parseProtobuf(r io.Reader, pb proto.Message) error {
	lenBytes := make([]byte, 4)
	err := readUntilCompleteOrError(r, lenBytes)
	if err != nil {
		glog.Errorf("failed to parse protobuf length with error %v", err)
		return err
	}
	pbLen := binary.BigEndian.Uint32(lenBytes)
	pbBytes := make([]byte, pbLen)
	err = readUntilCompleteOrError(r, pbBytes)
	if err != nil {
		glog.Errorf("failed to parse protobuf with error %v", err)
		return err
	}
	pbStr := string(pbBytes)
	err = proto.UnmarshalText(pbStr, pb)
	if err != nil {
		glog.Errorf("failed to unmarshal protobuf message with error %v", err)
		return err
	}
	return nil
}

// readUntilCompleteOrError keeps reading from the given reader until len(buf) bytes have been read
// or an error is encountered. Read returning 0 bytes and a nil error is considered an error and
// will result in readUntilCompleteOrError returning an error.
func readUntilCompleteOrError(r io.Reader, buf []byte) error {
	totalRead := 0
	for totalRead < len(buf) {
		bytesRead, err := r.Read(buf[totalRead:])
		totalRead += bytesRead
		if err == io.EOF || (bytesRead == 0 && err == nil) {
			if totalRead < len(buf) {
				return errors.New(fmt.Sprintf("Invalid file format. Expected to be able to read %d bytes, only read %d bytes", len(buf), totalRead))
			}
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
}
