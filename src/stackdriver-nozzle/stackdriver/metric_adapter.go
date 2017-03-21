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
	"bytes"
	"fmt"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	EventTime time.Time
	Unit      string // TODO Should this be "1" if it's empty?
}

func (m *Metric) Hash() string {
	var b bytes.Buffer
	b.Write([]byte(m.Name))

	// Extract keys to a slice and sort it
	keys := make([]string, len(m.Labels), len(m.Labels))
	for k, _ := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.Write([]byte(k))
		b.Write([]byte(m.Labels[k]))
	}
	return b.String()
}

type MetricAdapter interface {
	PostMetrics([]Metric) error
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

func (ma *metricAdapter) PostMetrics(metrics []Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	projectName := path.Join("projects", ma.projectID)
	var timeSerieses []*monitoringpb.TimeSeries

	for _, metric := range metrics {
		ma.heartbeater.Increment("metrics.count")

		err := ma.ensureMetricDescriptor(metric)
		if err != nil {
			return err
		}

		metricType := path.Join("custom.googleapis.com", metric.Name)
		timeSeries := monitoringpb.TimeSeries{
			Metric: &metricpb.Metric{
				Type:   metricType,
				Labels: metric.Labels,
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
	err := ma.client.Post(request)
	if err != nil {
		ma.heartbeater.Increment("metrics.errors")
	}
	err = errors.Wrapf(err, "Request: %+v", request)
	return err
}

func (ma *metricAdapter) CreateMetricDescriptor(metric Metric) error {
	projectName := path.Join("projects", ma.projectID)
	metricType := path.Join("custom.googleapis.com", metric.Name)
	metricName := path.Join(projectName, "metricDescriptors", metricType)

	var labelDescriptors []*labelpb.LabelDescriptor
	for key := range metric.Labels {
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

func (ma *metricAdapter) ensureMetricDescriptor(metric Metric) error {
	if metric.Unit == "" {
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
