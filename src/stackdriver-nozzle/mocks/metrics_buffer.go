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

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"

type MetricsBuffer struct {
	PostedMetrics []messages.Metric
}

func (m *MetricsBuffer) PostMetricEvents(events []*messages.MetricEvent) {
	for _, event := range events {
		for _, metric := range event.Metrics {
			m.PostedMetrics = append(m.PostedMetrics, *metric)
		}
	}
}

func (m *MetricsBuffer) IsEmpty() bool {
	return true
}
