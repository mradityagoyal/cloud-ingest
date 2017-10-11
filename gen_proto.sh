# This script handles the compilation of protocol buffers into different
# languages.
protoc --python_out=webconsole/backend/ proto/tasks.proto
