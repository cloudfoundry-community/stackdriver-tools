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

import "expvar"

// Value reads an Int counter by name
func Value(name string) int {
	val := expvar.Get(name)
	intVal := val.(*expvar.Int)
	return int(intVal.Value())
}

// MapValue reads a Map counter by name, key
func MapValue(name, key string) int {
	val := expvar.Get(name)
	mapVal := val.(*expvar.Map)
	return int(mapVal.Get(key).(*expvar.Int).Value())
}

// Reset sets all registered Int counters to 0
func Reset() {
	expvar.Do(func(value expvar.KeyValue) {
		resetKey(&value)
	})
}

func resetKey(value *expvar.KeyValue) {
	if intVal, ok := value.Value.(*expvar.Int); ok {
		intVal.Set(0)
	} else if mapVal, ok := value.Value.(*expvar.Map); ok {
		mapVal.Do(func(value expvar.KeyValue) {
			resetKey(&value)
		})
	}
}
