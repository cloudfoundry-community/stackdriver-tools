/*
 * Copyright 2019 Google Inc.
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

package metricspipeline

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

type router struct {
	metricAdapter stackdriver.MetricAdapter
	logAdapter    stackdriver.LogAdapter
	logEvents     map[events.Envelope_EventType]bool
	metricEvents  map[events.Envelope_EventType]bool
}

// NewRouter provides a MetricAdapter that routes a given metric to
// Stackdriver Logging and Stackdriver Monitoring based on configuration
func NewRouter(metricAdapter stackdriver.MetricAdapter, metricEvents []events.Envelope_EventType, logAdapter stackdriver.LogAdapter, logEvents []events.Envelope_EventType) stackdriver.MetricAdapter {
	r := &router{metricAdapter: metricAdapter, logAdapter: logAdapter}

	r.metricEvents = make(map[events.Envelope_EventType]bool)
	for _, e := range metricEvents {
		r.metricEvents[e] = true
	}

	r.logEvents = make(map[events.Envelope_EventType]bool)
	for _, e := range logEvents {
		r.logEvents[e] = true
	}

	return r
}

func (r *router) PostMetrics(metrics []*messages.Metric) {
	var metricEvents []*messages.Metric
	for i := range metrics {
		if r.metricEvents[metrics[i].Type] {
			metricEvents = append(metricEvents, metrics[i])
		}

		if r.logEvents[metrics[i].Type] {
			log := &messages.Log{
				Labels:  metrics[i].Labels,
				Payload: metrics[i],
			}
			r.logAdapter.PostLog(log)
		}
	}

	if len(metricEvents) > 0 {
		r.metricAdapter.PostMetrics(metricEvents)
	}
}
