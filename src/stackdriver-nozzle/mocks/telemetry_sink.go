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

import (
	"expvar"
	"sync"

	"github.com/pkg/errors"
)

type TelemetrySink struct {
	init       []*expvar.KeyValue
	lastReport []*expvar.KeyValue

	mu sync.Mutex
}

func (ts *TelemetrySink) Init(val []*expvar.KeyValue) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.init != nil {
		panic(errors.New("Init called more than once"))
	}

	ts.init = val
}

func (ts *TelemetrySink) Report(val []*expvar.KeyValue) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.lastReport = val
}

func (ts *TelemetrySink) GetInit() []*expvar.KeyValue {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.init
}

func (ts *TelemetrySink) GetLastReport() []*expvar.KeyValue {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.lastReport
}
