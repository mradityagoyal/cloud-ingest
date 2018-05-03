#!/bin/sh
# This script generates the Go protocol buffers for the protos in this dir.
protoc --go_out=task_go_proto/ task.proto
