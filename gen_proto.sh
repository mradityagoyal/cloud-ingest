#!/bin/sh
# This script handles the compilation of protocol buffers into different
# languages. This script should be run from the root of the repository.
pbjs_rel_path="./webconsole/frontend/node_modules/protobufjs/bin/pbjs"
pbts_rel_path="./webconsole/frontend/node_modules/protobufjs/bin/pbts"
protoc --python_out=webconsole/backend/ --go_out=dcp/ proto/tasks.proto
protoc --go_out=. tests/perf/proto/*.proto
if ! [ -e "$pbjs_rel_path" ]
then
  echo "The pbjs binary was not found. Did you install the web frontend dependencies?"
else
  $pbjs_rel_path -t static-module -w commonjs -o ./webconsole/frontend/src/app/proto/tasks.js ./proto/tasks.proto
  $pbts_rel_path -o ./webconsole/frontend/src/app/proto/tasks.d.ts ./webconsole/frontend/src/app/proto/tasks.js
fi
