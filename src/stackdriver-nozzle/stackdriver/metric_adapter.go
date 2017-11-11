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
	"time"

	"expvar"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry/lager"
	"github.com/golang/protobuf/ptypes/timestamp"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type MetricAdapter interface {
	PostMetricEvents([]*messages.MetricEvent)
}

var (
	timeSeriesReqs  *expvar.Int
	timeSeriesCount *expvar.Int
	timeSeriesErrs  *expvar.Map

	timeSeriesErrOutOfOrder *expvar.Int
	timeSeriesErrUnknown    *expvar.Int

	eventsCount *expvar.Int

	descriptorReqs *expvar.Int
	descriptorErrs *expvar.Int
)

func init() {
	timeSeriesReqs = expvar.NewInt("nozzle.metrics.timeseries.requests")
	timeSeriesCount = expvar.NewInt("nozzle.metrics.timeseries.count")
	timeSeriesErrs = expvar.NewMap("nozzle.metrics.timeseries.errors")

	timeSeriesErrOutOfOrder = &expvar.Int{}
	timeSeriesErrUnknown = &expvar.Int{}

	timeSeriesErrs.Set("out_of_order", timeSeriesErrOutOfOrder)
	timeSeriesErrs.Set("unknown", timeSeriesErrUnknown)

	eventsCount = expvar.NewInt("nozzle.metrics.firehose_events.count")

	descriptorReqs = expvar.NewInt("nozzle.metrics.descriptor.requests")
	descriptorErrs = expvar.NewInt("nozzle.metrics.descriptor.errors")
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

func (ma *metricAdapter) PostMetricEvents(events []*messages.MetricEvent) {
	series := ma.buildTimeSeries(events)
	projectName := path.Join("projects", ma.projectID)

	count := len(series)
	chunks := int(math.Ceil(float64(count) / float64(ma.batchSize)))

	ma.logger.Info("metricAdapter.PostMetricEvents", lager.Data{"info": "Posting TimeSeries to Stackdriver", "count": count, "chunks": chunks})
	var low, high int
	for i := 0; i < chunks; i++ {
		low = i * ma.batchSize
		high = low + ma.batchSize
		// if we're at the last chunk, take the remaining size
		// so we don't over index
		if i == chunks-1 {
			high = count
		}

		timeSeriesReqs.Add(1)
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
				timeSeriesErrOutOfOrder.Add(1)
			} else {
				timeSeriesErrUnknown.Add(1)
				ma.logger.Error("metricAdapter.PostMetricEvents", err, lager.Data{"info": "Unexpected Error", "request": request})
			}
		}
	}

	return
}

func (ma *metricAdapter) buildTimeSeries(events []*messages.MetricEvent) []*monitoringpb.TimeSeries {
	var timeSerieses []*monitoringpb.TimeSeries

	for _, event := range events {
		if len(event.Metrics) == 0 {
			continue
		}

		eventsCount.Add(1)
		for _, metric := range event.Metrics {
			timeSeriesCount.Add(1)
			err := ma.ensureMetricDescriptor(metric, event.Labels)
			if err != nil {
				ma.logger.Error("metricAdapter.buildTimeSeries", err, lager.Data{"metric": metric, "labels": event.Labels})
				continue
			}

			metricType := path.Join("custom.googleapis.com", metric.Name)
			timeSeries := monitoringpb.TimeSeries{
				Metric: &metricpb.Metric{
					Type:   metricType,
					Labels: event.Labels,
				},
				Points: points(metric.Value, metric.EventTime),
			}
			timeSerieses = append(timeSerieses, &timeSeries)
		}
	}

	return timeSerieses
}

func (ma *metricAdapter) CreateMetricDescriptor(metric *messages.Metric, labels map[string]string) error {
	projectName := path.Join("projects", ma.projectID)
	metricType := path.Join("custom.googleapis.com", metric.Name)
	metricName := path.Join(projectName, "metricDescriptors", metricType)

	var labelDescriptors []*labelpb.LabelDescriptor
	for key := range labels {
		labelDescriptors = append(labelDescriptors, &labelpb.LabelDescriptor{
			Key:       key,
			ValueType: labelpb.LabelDescriptor_STRING,
		})
	}

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name: projectName,
		MetricDescriptor: &metricpb.MetricDescriptor{
			Name:        metricName,
			Type:        metricType,
			Labels:      labelDescriptors,
			MetricKind:  metricpb.MetricDescriptor_GAUGE,
			ValueType:   metricpb.MetricDescriptor_DOUBLE,
			Unit:        metric.Unit,
			Description: "stackdriver-nozzle created custom metric.",
			DisplayName: metric.Name, // TODO
		},
	}

	descriptorReqs.Add(1)
	if err := ma.client.CreateMetricDescriptor(req); err != nil {
		descriptorErrs.Add(1)
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

func (ma *metricAdapter) ensureMetricDescriptor(metric *messages.Metric, labels map[string]string) error {
	if metric.Unit == "" {
		return nil
	}

	ma.createDescriptorMutex.Lock()
	defer ma.createDescriptorMutex.Unlock()

	if _, ok := ma.descriptors[metric.Name]; ok {
		return nil
	}

	err := ma.CreateMetricDescriptor(metric, labels)
	if err != nil {
		return err
	}
	ma.descriptors[metric.Name] = struct{}{}
	return nil
}

func points(value float64, eventTime time.Time) []*monitoringpb.Point {
	timeStamp := timestamp.Timestamp{
		Seconds: eventTime.Unix(),
		Nanos:   int32(eventTime.Nanosecond()),
	}
	point := &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &timeStamp,
			StartTime: &timeStamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
	return []*monitoringpb.Point{point}
}
