/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package telemetrytest

import (
	"expvar"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
)

// Counter reads an Int counter by name
func Counter(name string) int {
	val := telemetry.Get(name)
	intVal := val.(*telemetry.Counter)
	return int(intVal.Value())
}

// MapCounter reads a Counter from a MapCounter by name, key
func MapCounter(name, key string) int {
	val := telemetry.Get(name)
	mapVal := val.(*telemetry.CounterMap)
	return int(mapVal.Get(key).(*telemetry.Counter).Value())
}

// Reset sets all registered Int counters to 0
func Reset() {
	telemetry.Do(func(value expvar.KeyValue) {
		resetKey(&value)
	})
}

func resetKey(value *expvar.KeyValue) {
	if intVal, ok := value.Value.(*telemetry.Counter); ok {
		intVal.Set(0)
	} else if mapVal, ok := value.Value.(*telemetry.CounterMap); ok {
		mapVal.Do(func(value expvar.KeyValue) {
			resetKey(&value)
		})
	}
}
