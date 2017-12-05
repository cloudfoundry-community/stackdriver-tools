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
	"strings"
	"sync"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/lager"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type MetricAdapter interface {
	PostMetrics([]*messages.Metric)
}

var (
	timeSeriesReqs  *telemetry.Counter
	timeSeriesCount *telemetry.Counter
	timeSeriesErrs  *telemetry.CounterMap

	timeSeriesErrOutOfOrder *telemetry.Counter
	timeSeriesErrUnknown    *telemetry.Counter

	firehoseEventsCount *telemetry.Counter

	descriptorReqs *telemetry.Counter
	descriptorErrs *telemetry.Counter
)

func init() {
	timeSeriesReqs = telemetry.NewCounter(telemetry.Nozzle, "metrics.timeseries.requests")
	timeSeriesCount = telemetry.NewCounter(telemetry.Nozzle, "metrics.timeseries.count")
	timeSeriesErrs = telemetry.NewCounterMap(telemetry.Nozzle, "metrics.timeseries.errors", "error_type")

	timeSeriesErrOutOfOrder = &telemetry.Counter{}
	timeSeriesErrUnknown = &telemetry.Counter{}

	timeSeriesErrs.Set("out_of_order", timeSeriesErrOutOfOrder)
	timeSeriesErrs.Set("unknown", timeSeriesErrUnknown)

	firehoseEventsCount = telemetry.NewCounter(telemetry.Nozzle, "metrics.firehose_events.emitted.count")

	descriptorReqs = telemetry.NewCounter(telemetry.Nozzle, "metrics.descriptor.requests")
	descriptorErrs = telemetry.NewCounter(telemetry.Nozzle, "metrics.descriptor.errors")
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
		request := &monitoringpb.CreateTimeSeriesRequest{
			Name:       projectName,
			TimeSeries: series[low:high],
		}
		err := ma.client.Post(request)

		if err != nil {
			// This is an expected error once there is more than a single nozzle writing to Stackdriver.
			// If one nozzle writes a metric occurring at time T2 and this one tries to write at T1 (where T2 later than T1)
			// we will receive this error.
			if strings.Contains(err.Error(), `Points must be written in order`) {
				timeSeriesErrOutOfOrder.Increment()
			} else {
				timeSeriesErrUnknown.Increment()
				ma.logger.Error("metricAdapter.PostMetrics", err, lager.Data{"info": "Unexpected Error", "request": request})
			}
		}
	}

	return
}

func (ma *metricAdapter) buildTimeSeries(metrics []*messages.Metric) []*monitoringpb.TimeSeries {
	var timeSerieses []*monitoringpb.TimeSeries

	for _, metric := range metrics {
		err := ma.ensureMetricDescriptor(metric)
		if err != nil {
			ma.logger.Error("metricAdapter.buildTimeSeries", err, lager.Data{"metric": metric})
			continue
		}

		firehoseEventsCount.Increment()
		timeSeriesCount.Increment()
		timeSerieses = append(timeSerieses, metric.TimeSeries())
	}

	return timeSerieses
}

func (ma *metricAdapter) CreateMetricDescriptor(metric *messages.Metric) error {
	projectName := path.Join("projects", ma.projectID)

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name:             projectName,
		MetricDescriptor: metric.MetricDescriptor(projectName),
	}

	descriptorReqs.Increment()
	if err := ma.client.CreateMetricDescriptor(req); err != nil {
		descriptorErrs.Increment()
		return err
	}

	return nil
}

func (ma *metricAdapter) fetchMetricDescriptorNames() error {
	req := &monitoringpb.ListMetricDescriptorsRequest{
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
