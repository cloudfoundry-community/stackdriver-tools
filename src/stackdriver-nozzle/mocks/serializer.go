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
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry/sonde-go/events"
)

type MockSerializer struct {
	GetLogFn     func(*events.Envelope) *messages.Log
	GetMetricsFn func(*events.Envelope) ([]messages.DataPoint, error)
	IsLogFn      func(*events.Envelope) bool
}

func (m *MockSerializer) GetLog(envelope *events.Envelope) *messages.Log {
	if m.GetLogFn != nil {
		return m.GetLogFn(envelope)
	}
	return nil
}

func (m *MockSerializer) GetMetrics(envelope *events.Envelope) ([]messages.DataPoint, error) {
	if m.GetMetricsFn != nil {
		return m.GetMetricsFn(envelope)
	}
	return nil, nil
}

func (m *MockSerializer) IsLog(envelope *events.Envelope) bool {
	if m.IsLogFn != nil {
		return m.IsLogFn(envelope)
	}
	return true
}
