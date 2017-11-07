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

package heartbeat

import (
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/lager"
)

type metricHandler struct {
	start      time.Time
	logger     lager.Logger
	ma         stackdriver.MetricAdapter
	nozzleId   string
	nozzleName string
	nozzleZone string

	counterMu *sync.Mutex // Guards counter
	counter   map[string]uint
}

func NewMetricHandler(ma stackdriver.MetricAdapter, logger lager.Logger, nozzleId, nozzleName, nozzleZone string) *metricHandler {
	return &metricHandler{
		logger:     logger,
		ma:         ma,
		nozzleId:   nozzleId,
		nozzleName: nozzleName,
		nozzleZone: nozzleZone,
		start:      time.Now(),
		counterMu:  &sync.Mutex{},
		counter:    map[string]uint{},
	}
}

func (h *metricHandler) Name() string {
	return "metricHandler"
}

func (h *metricHandler) Handle(event string, count uint) {
	h.counterMu.Lock()
	defer h.counterMu.Unlock()
	h.counter[event] += count
	return
}

func (h *metricHandler) Flush() {
	counter := h.flushInternal()

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

func (h *metricHandler) flushInternal() map[string]uint {
	h.counterMu.Lock()
	defer h.counterMu.Unlock()

	counter := h.counter
	h.counter = map[string]uint{}
	return counter
}
