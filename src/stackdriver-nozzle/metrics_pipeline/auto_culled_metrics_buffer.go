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

package metrics_pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/lager"
)

type autoCulledMetricsBuffer struct {
	adapter     stackdriver.MetricAdapter
	errs        chan error
	ticker      *time.Ticker
	ctx         context.Context
	logger      lager.Logger
	heartbeater heartbeat.Heartbeater

	metricsMu sync.Mutex // Guard metrics
	metrics   map[string]*messages.MetricEvent
}

// NewAutoCulledMetricsBuffer provides a MetricsBuffer that will cull like metrics over the defined frequency.
// A like metric is defined as a metric with the same stackdriver.Metric.Hash()
func NewAutoCulledMetricsBuffer(ctx context.Context, logger lager.Logger, frequency time.Duration,
	adapter stackdriver.MetricAdapter, heartbeater heartbeat.Heartbeater) (MetricsBuffer, <-chan error) {
	errs := make(chan error)
	mb := &autoCulledMetricsBuffer{
		adapter:     adapter,
		errs:        errs,
		metrics:     make(map[string]*messages.MetricEvent),
		ctx:         ctx,
		logger:      logger,
		ticker:      time.NewTicker(frequency),
		heartbeater: heartbeater,
	}
	mb.start()
	return mb, errs
}

func (mb *autoCulledMetricsBuffer) PostMetricEvents(events []*messages.MetricEvent) error {
	mb.metricsMu.Lock()
	defer mb.metricsMu.Unlock()

	for _, event := range events {
		hash := event.Hash()
		old, exists := mb.metrics[hash]
		if !exists {
			mb.metrics[hash] = event
		} else {
			mb.heartbeater.Increment("metrics.events.sampled")
			if event.Metrics[0].EventTime.After(old.Metrics[0].EventTime) {
				// Firehose messages are not guaranteed to be received in
				// timestamp order, so only overwrite the sampled metric
				// if the event is newer.
				mb.metrics[hash] = event
			}
		}
	}

	return nil
}

func (mb *autoCulledMetricsBuffer) IsEmpty() bool {
	return len(mb.metrics) == 0
}

func (mb *autoCulledMetricsBuffer) flush() {
	err := mb.adapter.PostMetricEvents(mb.flushInternalBuffer())

	if err != nil {
		mb.errs <- err
	}
}

func (mb *autoCulledMetricsBuffer) flushInternalBuffer() []*messages.MetricEvent {
	mb.metricsMu.Lock()
	defer mb.metricsMu.Unlock()
	mb.logger.Info("autoCulledMetricsBuffer", lager.Data{"info": fmt.Sprintf("Flushing %v metrics", len(mb.metrics))})

	events := make([]*messages.MetricEvent, 0, len(mb.metrics))
	for _, v := range mb.metrics {
		events = append(events, v)
	}

	mb.metrics = make(map[string]*messages.MetricEvent)

	return events
}

func (mb *autoCulledMetricsBuffer) start() {
	go func() {
		defer close(mb.errs)
		for {
			select {
			case <-mb.ticker.C:
				mb.flush()
			case <-mb.ctx.Done():
				mb.logger.Info("autoCulledMetricsBuffer", lager.Data{"info": "Context cancelled"})
				mb.ticker.Stop()
				mb.flush()
				return
			}
		}
	}()
}
