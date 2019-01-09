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
	"sync"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
)

type MetricAdapter struct {
	sync.Mutex

	PostMetricsFn    func(metrics []*messages.Metric) error
	PostMetricsCount int
	PostedMetrics    []*messages.Metric
}

func (m *MetricAdapter) PostMetrics(metrics []*messages.Metric) {
	m.Lock()
	defer m.Unlock()

	m.PostMetricsCount++

	if m.PostMetricsFn != nil {
		if err := m.PostMetricsFn(metrics); err != nil {
			panic(err)
		}
	}

	m.PostedMetrics = append(m.PostedMetrics, metrics...)
}

func (m *MetricAdapter) GetPostedMetrics() []*messages.Metric {
	m.Lock()
	defer m.Unlock()

	return m.PostedMetrics
}
