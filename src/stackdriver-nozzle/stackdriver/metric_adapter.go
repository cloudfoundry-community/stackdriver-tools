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
	"path"
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type MetricAdapter interface {
	PostMetricEvents([]*messages.MetricEvent) error
}

type Heartbeater interface {
	Start()
	Increment(string)
	Stop()
}

type metricAdapter struct {
	projectID             string
	client                MetricClient
	descriptors           map[string]struct{}
	createDescriptorMutex *sync.Mutex
	heartbeater           Heartbeater
}

// NewMetricAdapter returns a MetricAdapater that can write to Stackdriver Monitoring
func NewMetricAdapter(projectID string, client MetricClient, heartbeater Heartbeater) (MetricAdapter, error) {
	ma := &metricAdapter{
		projectID:             projectID,
		client:                client,
		createDescriptorMutex: &sync.Mutex{},
		descriptors:           map[string]struct{}{},
		heartbeater:           heartbeater,
	}

	err := ma.fetchMetricDescriptorNames()
	return ma, err
}

func (ma *metricAdapter) PostMetricEvents(events []*messages.MetricEvent) (err error) {
	for _, event := range events {
		if len(event.Metrics) == 0 {
			return nil
		}

		projectName := path.Join("projects", ma.projectID)
		var timeSerieses []*monitoringpb.TimeSeries
		ma.heartbeater.Increment("metrics.events.count")

		for _, metric := range event.Metrics {
			ma.heartbeater.Increment("metrics.count")
			err := ma.ensureMetricDescriptor(metric, event.Labels)
			if err != nil {
				return err
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

		request := &monitoringpb.CreateTimeSeriesRequest{
			Name:       projectName,
			TimeSeries: timeSerieses,
		}

		ma.heartbeater.Increment("metrics.requests")
		err = ma.client.Post(request)
		if err != nil {
			ma.heartbeater.Increment("metrics.errors")
		}
		err = errors.Wrapf(err, "Request: %+v", request)
	}
	return
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

	return ma.client.CreateMetricDescriptor(req)
}

func (ma *metricAdapter) fetchMetricDescriptorNames() error {
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", ma.projectID),
		Filter: "metric.type = starts_with(\"custom.googleapis.com/\")",
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
