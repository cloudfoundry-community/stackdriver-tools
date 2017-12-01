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

type Counter struct {
	expvar.Int
}

func (c *Counter) Increment() {
	c.Add(1)
}

// IntValue returns the counter's value as an int rather than an int64. Tests are
// generally written with int types, so this is useful to avoid scattering type
// casts around the test codebase.
func (c *Counter) IntValue() int {
	return int(c.Value())
}

type CounterMap struct {
	expvar.Map
	category string
}

func (cm *CounterMap) Category() string {
	return cm.category
}

// The metricSet contains a map of metric prefixes and enables
// the creation of new prefixes which are automatically exported.
var metricSet = struct {
	prefixes map[MetricPrefix][]string
	mu       sync.Mutex
}{
	prefixes: map[MetricPrefix][]string{},
}

// A MetricPrefix is a path element prepended to metric names.
type MetricPrefix string

// Nozzle is the prefix under which the nozzle exports metrics about
// its own operation. It's created here because metrics will be created
// in many places throughout the Nozzle's code base.
const Nozzle MetricPrefix = "stackdriver-nozzle"

// Qualify returns the metric name prepended by the metric prefix and "/".
func (mp MetricPrefix) Qualify(name string) string {
	return string(mp) + "/" + name
}

// NewCounter creates and exports a new Counter for the MetricPrefix.
func NewCounter(mp MetricPrefix, name string) *Counter {
	v := new(Counter)
	publish(mp, name, v)
	return v
}

// NewCounterMap creates and exports a new CounterMap for the MetricPrefix.
func NewCounterMap(mp MetricPrefix, name, category string) *CounterMap {
	v := new(CounterMap)
	v.category = category

	publish(mp, name, v)
	return v
}

func publish(mp MetricPrefix, name string, v expvar.Var) {
	metricSet.mu.Lock()
	defer metricSet.mu.Unlock()
	if _, ok := metricSet.prefixes[mp]; !ok {
		metricSet.prefixes[mp] = []string{name}
	} else {
		metricSet.prefixes[mp] = append(metricSet.prefixes[mp], name)
	}
	expvar.Publish(mp.Qualify(name), v)
}

// forEachMetric calls f for each exported variable.
// The global metric set is locked during the iteration,
// but existing entries may be concurrently updated.
func forEachMetric(f func(expvar.KeyValue)) {
	metricSet.mu.Lock()
	defer metricSet.mu.Unlock()
	for mp, counters := range metricSet.prefixes {
		for _, k := range counters {
			val := Get(mp, k)
			f(expvar.KeyValue{Key: k, Value: val.(expvar.Var)})
		}
	}
}

// Get retrieves a named exported variable with the given MetricPrefix.
// It returns nil if the name has not been registered.
func Get(mp MetricPrefix, name string) expvar.Var {
	return expvar.Get(mp.Qualify(name))
}
