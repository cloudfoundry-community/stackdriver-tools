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
	"context"
	"sync"
	"time"
)

type autoCulledMetricsBuffer struct {
	adapter MetricAdapter
	errs    chan error
	size    int
	ticker  *time.Ticker
	ctx     context.Context

	metricsMu sync.Mutex // Guard metrics
	metrics   map[string]*Metric
}

func NewAutoCulledMetricsBuffer(ctx context.Context, frequency time.Duration,
	size int, adapter MetricAdapter) (MetricsBuffer, <-chan error) {
	errs := make(chan error)
	mb := &autoCulledMetricsBuffer{
		adapter: adapter,
		errs:    errs,
		metrics: make(map[string]*Metric),
		size:    size,
		ctx:     ctx,
		ticker:  time.NewTicker(frequency),
	}
	mb.start()
	return mb, errs
}

func (mb *autoCulledMetricsBuffer) PostMetric(metric *Metric) {
	mb.addMetric(metric)
}

func (mb *autoCulledMetricsBuffer) IsEmpty() bool {
	return len(mb.metrics) == 0
}

func (mb *autoCulledMetricsBuffer) addMetric(newMetric *Metric) {
	mb.metricsMu.Lock()
	defer mb.metricsMu.Unlock()
	mb.metrics[newMetric.Hash()] = newMetric
}

func (mb *autoCulledMetricsBuffer) start() {
	go func() {
		for {
			select {
			case <-mb.ticker.C:
				mb.metricsMu.Lock()
				metricsSlice := metricsMapToSlice(mb.metrics)
				l := len(metricsSlice)
				chunks := l/mb.size + 1
				var low, high int
				for i := 0; i < chunks; i++ {
					low = i * mb.size
					high = low + mb.size
					if i == chunks-1 {
						high = l
					}
					err := mb.adapter.PostMetrics(metricsSlice[low:high])

					if err != nil {
						mb.errs <- err
					}
				}
				mb.metrics = make(map[string]*Metric)
				mb.metricsMu.Unlock()

			case <-mb.ctx.Done():
				mb.ticker.Stop()
				mb.metricsMu.Lock()
				err := mb.adapter.PostMetrics(metricsMapToSlice(mb.metrics))
				mb.metricsMu.Unlock()
				if err != nil {
					mb.errs <- err
				}
				return
			}
		}
	}()
}
