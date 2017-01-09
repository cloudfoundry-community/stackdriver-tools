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

import "reflect"

type MetricsBuffer interface {
	PostMetric(*Metric)
}

type metricsBuffer struct {
	size    int
	adapter MetricAdapter
	errs    chan error
	metrics []Metric
}

func NewMetricsBuffer(size int, adapter MetricAdapter) (MetricsBuffer, <-chan error) {
	errs := make(chan error)
	return &metricsBuffer{size, adapter, errs, []Metric{}}, errs
}

func (mb *metricsBuffer) PostMetric(metric *Metric) {
	mb.addMetric(metric)

	if len(mb.metrics) < mb.size {
		return
	}

	mb.postMetrics(mb.metrics)
	mb.metrics = []Metric{}
}

func (mb *metricsBuffer) addMetric(newMetric *Metric) {
	var existingMetric *Metric

	for _, metric := range mb.metrics {
		if metric.Name == newMetric.Name &&
			reflect.DeepEqual(metric.Labels, newMetric.Labels) {
			existingMetric = &metric
			break
		}
	}

	if existingMetric == nil {
		mb.metrics = append(mb.metrics, *newMetric)
	} else {
		/*
			Stack driver API does not let us have multiple time series with the same name/label
			in a single request. Furthermore, within each time series, we cannot have multiple points.
			Due to this, if we encounter a metric with same name/labels, we will send the current buffer
			and make a new buffer with the duplicate metric (╯°□°）╯︵ ┻━┻
		*/
		mb.postMetrics(mb.metrics)
		mb.metrics = []Metric{*newMetric}
	}
}

func (mb *metricsBuffer) postMetrics(metrics []Metric) {
	go func() {
		err := mb.adapter.PostMetrics(metrics)
		if err != nil {
			mb.errs <- err
		}
	}()
}
