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

package telemetry

import (
	"expvar"
	"sync"
)

const prefix = "stackdriver-nozzle/"

type Counter struct {
	expvar.Int
}

func (c *Counter) Increment() {
	c.Add(1)
}

type CounterMap struct {
	expvar.Map
	category string
}

func (cm *CounterMap) Category() string {
	return cm.category
}

func NewCounter(name string) *Counter {
	v := new(Counter)
	publish(name, v)
	return v
}

func NewCounterMap(name, category string) *CounterMap {
	v := new(CounterMap)
	v.category = category

	publish(name, v)
	return v
}

var counters []string
var countersMu sync.Mutex

func publish(name string, v expvar.Var) {
	countersMu.Lock()
	defer countersMu.Unlock()

	counters = append(counters, name)
	expvar.Publish(prefix+name, v)
}

// Do calls f for each exported variable.
// The global counter map is locked during the iteration,
// but existing entries may be concurrently updated.
func Do(f func(expvar.KeyValue)) {
	countersMu.Lock()
	defer countersMu.Unlock()

	for _, k := range counters {
		val := Get(k)
		f(expvar.KeyValue{Key: k, Value: val.(expvar.Var)})
	}
}

// Get retrieves a named exported variable. It returns nil if the name has
// not been registered.
func Get(name string) expvar.Var {
	return expvar.Get(prefix + name)
}
