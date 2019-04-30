#!/bin/sh
# This script generates the Go protocol buffers for the protos in this dir.
protoc --go_out=$(go env GOPATH)/src/ task.proto
protoc --go_out=$(go env GOPATH)/src/ pulse.proto
protoc --go_out=$(go env GOPATH)/src/ control.proto
protoc --go_out=$(go env GOPATH)/src/ listfile.proto
