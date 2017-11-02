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
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/golang/protobuf/ptypes/timestamp"
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
	IncrementBy(string, uint)
	Stop()
}

type metricAdapter struct {
	projectID             string
	client                MetricClient
	descriptors           map[string]struct{}
	createDescriptorMutex *sync.Mutex
	heartbeater           Heartbeater
}

type postMetricErr struct {
	metricDescriptor []error
	postErr          error
	heartbeater      Heartbeater
}

func (pme *postMetricErr) BuildError() error {
	pme.processErrors()
	if pme == nil || pme.postErr == nil && len(pme.metricDescriptor) == 0 {
		return nil
	}

	return fmt.Errorf("PostMetricEvents, PostErr: %v, MetricDescriptor: %v", pme.postErr, pme.metricDescriptor)
}

func (pme *postMetricErr) processErrors() {
	if pme == nil || pme.postErr == nil && len(pme.metricDescriptor) == 0 {
		return
	}

	if pme.postErr != nil {
		pme.heartbeater.Increment("metrics.post.errors")

		// This is an expected error once there is more than a single nozzle writing to Stackdriver.
		// If one nozzle writes a metric occuring at time T2 and this one tries to write at T1 (where T2 later than T1)
		// we will receive this error.
		if strings.Contains(pme.postErr.Error(), `Points must be written in order`) {
			pme.heartbeater.Increment("metrics.post.errors.out_of_order")
			pme.postErr = nil
		} else {
			pme.heartbeater.Increment("metrics.post.errors.unknown")
		}
	}

	metricDescriptorErrs := len(pme.metricDescriptor)
	if metricDescriptorErrs > 0 {
		pme.heartbeater.IncrementBy("metrics.metric_descriptor.errors", uint(metricDescriptorErrs))
	}
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

func (ma *metricAdapter) PostMetricEvents(events []*messages.MetricEvent) error {
	series, compositeErr := ma.buildTimeSeries(events)

	projectName := path.Join("projects", ma.projectID)
	request := &monitoringpb.CreateTimeSeriesRequest{
		Name:       projectName,
		TimeSeries: series,
	}

	ma.heartbeater.Increment("metrics.requests")
	compositeErr.postErr = ma.client.Post(request)

	return compositeErr.BuildError()
}

func (ma *metricAdapter) buildTimeSeries(events []*messages.MetricEvent) ([]*monitoringpb.TimeSeries, postMetricErr) {
	var timeSerieses []*monitoringpb.TimeSeries

	compositeErr := postMetricErr{heartbeater: ma.heartbeater}

	for _, event := range events {
		if len(event.Metrics) == 0 {
			continue
		}

		ma.heartbeater.Increment("metrics.events.count")
		for _, metric := range event.Metrics {
			ma.heartbeater.Increment("metrics.timeseries.count")
			err := ma.ensureMetricDescriptor(metric, event.Labels)
			if err != nil {
				compositeErr.metricDescriptor = append(compositeErr.metricDescriptor, err)
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

	return timeSerieses, compositeErr
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
