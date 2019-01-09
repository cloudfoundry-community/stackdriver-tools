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
	"fmt"
	"math"
	"path"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"google.golang.org/genproto/googleapis/monitoring/v3"
)

type MetricAdapter interface {
	PostMetrics([]*messages.Metric)
}

var (
	timeSeriesCount *telemetry.Counter
)

func init() {
	timeSeriesCount = telemetry.NewCounter(telemetry.Nozzle, "metrics.timeseries.count")
}

type metricAdapter struct {
	projectID             string
	client                MetricClient
	descriptors           map[string]struct{}
	createDescriptorMutex *sync.Mutex
	batchSize             int
	logger                lager.Logger
}

// NewMetricAdapter returns a MetricAdapater that can write to Stackdriver Monitoring
func NewMetricAdapter(projectID string, client MetricClient, batchSize int, logger lager.Logger) (MetricAdapter, error) {
	ma := &metricAdapter{
		projectID:             projectID,
		client:                client,
		createDescriptorMutex: &sync.Mutex{},
		descriptors:           map[string]struct{}{},
		batchSize:             batchSize,
		logger:                logger,
	}

	err := ma.fetchMetricDescriptorNames()
	return ma, err
}

func (ma *metricAdapter) PostMetrics(metrics []*messages.Metric) {
	series := ma.buildTimeSeries(metrics)
	projectName := path.Join("projects", ma.projectID)

	count := len(series)
	chunks := int(math.Ceil(float64(count) / float64(ma.batchSize)))

	ma.logger.Info("metricAdapter.PostMetrics", lager.Data{"info": "Posting TimeSeries to Stackdriver", "count": count, "chunks": chunks})
	var low, high int
	for i := 0; i < chunks; i++ {
		low = i * ma.batchSize
		high = low + ma.batchSize
		// if we're at the last chunk, take the remaining size
		// so we don't over index
		if i == chunks-1 {
			high = count
		}

		timeSeriesReqs.Increment()
		request := &monitoring.CreateTimeSeriesRequest{
			Name:       projectName,
			TimeSeries: series[low:high],
		}

		if err := ma.client.Post(request); err != nil {
			ma.logger.Error("metricAdapter.PostMetrics", err, lager.Data{"info": "Unexpected Error", "request": request})
		}
	}
}

func (ma *metricAdapter) buildTimeSeries(metrics []*messages.Metric) []*monitoring.TimeSeries {
	var timeSerieses []*monitoring.TimeSeries

	for _, metric := range metrics {
		err := ma.ensureMetricDescriptor(metric)
		if err != nil {
			ma.logger.Error("metricAdapter.buildTimeSeries", err, lager.Data{"metric": metric})
			continue
		}

		timeSeriesCount.Increment()
		timeSerieses = append(timeSerieses, metric.TimeSeries())
	}

	return timeSerieses
}

func (ma *metricAdapter) CreateMetricDescriptor(metric *messages.Metric) error {
	projectName := path.Join("projects", ma.projectID)

	req := &monitoring.CreateMetricDescriptorRequest{
		Name:             projectName,
		MetricDescriptor: metric.MetricDescriptor(projectName),
	}

	return ma.client.CreateMetricDescriptor(req)
}

func (ma *metricAdapter) fetchMetricDescriptorNames() error {
	req := &monitoring.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", ma.projectID),
		Filter: `metric.type = starts_with("custom.googleapis.com/")`,
	}

	descriptors, err := ma.client.ListMetricDescriptors(req)
	if err != nil {
		return err
	}

	for _, descriptor := range descriptors {
		ma.descriptors[descriptor.Name] = struct{}{}
	}
	return nil
}

func (ma *metricAdapter) ensureMetricDescriptor(metric *messages.Metric) error {
	if !metric.NeedsMetricDescriptor() {
		return nil
	}

	ma.createDescriptorMutex.Lock()
	defer ma.createDescriptorMutex.Unlock()

	if _, ok := ma.descriptors[metric.Name]; ok {
		return nil
	}

	err := ma.CreateMetricDescriptor(metric)
	if err != nil {
		return err
	}
	ma.descriptors[metric.Name] = struct{}{}
	return nil
}
