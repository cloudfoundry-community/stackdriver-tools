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

package stackdriver

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/lager"
)

type telemetrySink struct {
	start      time.Time
	logger     lager.Logger
	ma         MetricAdapter
	nozzleId   string
	nozzleName string
	nozzleZone string
}

func NewTelemetrySink(ma MetricAdapter, logger lager.Logger, nozzleId, nozzleName, nozzleZone string) telemetry.Sink {
	return &telemetrySink{
		logger:     logger,
		ma:         ma,
		nozzleId:   nozzleId,
		nozzleName: nozzleName,
		nozzleZone: nozzleZone,
		start:      time.Now(),
	}
}

func (h *telemetrySink) Record(counter map[string]int) {
	metrics := []*messages.Metric{}
	t := time.Now()
	labels := map[string]string{
		"instance": h.nozzleName,
		"zone":     h.nozzleZone,
	}

	for k, v := range counter {
		metrics = append(metrics, &messages.Metric{
			Name:      "heartbeat." + k,
			Value:     float64(v),
			EventTime: t,
		})
	}
	h.ma.PostMetricEvents([]*messages.MetricEvent{{Labels: labels, Metrics: metrics}})
}
