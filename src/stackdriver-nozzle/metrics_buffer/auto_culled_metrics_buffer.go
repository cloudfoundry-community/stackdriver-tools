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

package metrics_buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/lager"
)

type autoCulledMetricsBuffer struct {
	adapter stackdriver.MetricAdapter
	errs    chan error
	size    int
	ticker  *time.Ticker
	ctx     context.Context
	logger  lager.Logger

	metricsMu sync.Mutex // Guard metrics
	metrics   map[string]*messages.Metric
}

// NewAutoCulledMetricsBuffer provides a MetricsBuffer that will cull like metrics over the defined frequency.
// A like metric is defined as a metric with the same stackdriver.Metric.Hash()
func NewAutoCulledMetricsBuffer(ctx context.Context, logger lager.Logger, frequency time.Duration,
	size int, adapter stackdriver.MetricAdapter) (MetricsBuffer, <-chan error) {
	errs := make(chan error)
	mb := &autoCulledMetricsBuffer{
		adapter: adapter,
		errs:    errs,
		metrics: make(map[string]*messages.Metric),
		size:    size,
		ctx:     ctx,
		logger:  logger,
		ticker:  time.NewTicker(frequency),
	}
	mb.start()
	return mb, errs
}

func (mb *autoCulledMetricsBuffer) PostMetric(metric *messages.Metric) {
	mb.addMetric(metric)
}

func (mb *autoCulledMetricsBuffer) PostMetrics(metrics []messages.Metric) error {
	for i, _ := range metrics {
		metric := metrics[i]
		mb.addMetric(&metric)
	}

	return nil
}

func (mb *autoCulledMetricsBuffer) IsEmpty() bool {
	return len(mb.metrics) == 0
}

func (mb *autoCulledMetricsBuffer) addMetric(newMetric *messages.Metric) {
	mb.metricsMu.Lock()
	defer mb.metricsMu.Unlock()
	mb.metrics[newMetric.Hash()] = newMetric
}

func (mb *autoCulledMetricsBuffer) flush() {
	metrics := mb.flushInternalBuffer()
	count := len(metrics)
	chunks := count/mb.size + 1

	mb.logger.Info("autoCulledMetricsBuffer", lager.Data{"info": fmt.Sprintf("%v metrics will be flushed in %v batches", count, chunks)})
	var low, high int
	for i := 0; i < chunks; i++ {
		low = i * mb.size
		high = low + mb.size
		if i == chunks-1 {
			high = count
		}
		err := mb.adapter.PostMetrics(metrics[low:high])

		if err != nil {
			mb.errs <- err
		}
	}
}

func (mb *autoCulledMetricsBuffer) flushInternalBuffer() []messages.Metric {
	mb.metricsMu.Lock()
	defer mb.metricsMu.Unlock()
	mb.logger.Info("autoCulledMetricsBuffer", lager.Data{"info": fmt.Sprintf("Flushing %v metrics", len(mb.metrics))})

	metrics := make([]messages.Metric, 0, len(mb.metrics))
	for _, v := range mb.metrics {
		metrics = append(metrics, *v)
	}

	mb.metrics = make(map[string]*messages.Metric)

	return metrics
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
