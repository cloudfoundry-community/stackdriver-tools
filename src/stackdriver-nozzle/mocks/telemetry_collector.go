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

package mocks

import "sync"

func NewCollector() *Collector {
	return &Collector{counters: map[string]int{}}
}

type Collector struct {
	started  bool
	counters map[string]int
	mutex    sync.Mutex
}

func (h *Collector) Start() {
	h.mutex.Lock()
	h.started = true
	h.mutex.Unlock()
}

func (h *Collector) Increment(name string) {
	h.IncrementBy(name, 1)
}

func (h *Collector) IncrementBy(name string, count int) {
	h.mutex.Lock()
	h.counters[name] += int(count)
	h.mutex.Unlock()
}

func (h *Collector) Stop() {
	h.mutex.Lock()
	h.started = false
	h.mutex.Unlock()
}

func (h *Collector) IsRunning() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.started
}

func (h *Collector) GetCount(name string) int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.counters[name]
}
