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

package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type LogFields map[string]interface{}

func (lf LogFields) val(key string) int64 {
	value, err := lf[key].(json.Number).Int64()
	if err != nil {
		return int64(0)
	}
	return value
}

func (lf LogFields) String() string {
	var keys []string
	for k, _ := range lf {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var kv []string
	for _, k := range keys {
		kv = append(kv, fmt.Sprintf("%v:%v", k, lf[k]))
	}
	return strings.Join(kv, " ")
}
