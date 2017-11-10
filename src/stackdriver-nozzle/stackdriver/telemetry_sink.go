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
	"expvar"
	"fmt"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/lager"
	"github.com/golang/protobuf/ptypes/timestamp"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type telemetrySink struct {
	projectPath string
	labels      map[string]string
	resource    *monitoredres.MonitoredResource
	logger      lager.Logger
	client      MetricClient
	startTime   *timestamp.Timestamp
}

func now() *timestamp.Timestamp {
	now := time.Now()
	return &timestamp.Timestamp{
		Seconds: now.Unix(),
		Nanos:   int32(now.Nanosecond()),
	}
}

func detectMonitoredResource() (res *monitoredres.MonitoredResource) {
	res = &monitoredres.MonitoredResource{Type: "global"}

	if metadata.OnGCE() {
		projectId, err := metadata.ProjectID()
		if err != nil {
			return
		}
		instanceId, err := metadata.InstanceID()
		if err != nil {
			return
		}
		zone, err := metadata.Zone()
		if err != nil {
			return
		}

		res.Type = "gce_instance"
		res.Labels = map[string]string{"project_id": projectId, "instance_id": instanceId, "zone": zone}
	}
	return
}

// NewTelemetrySink provides a telemetry.Sink that writes metrics to Stackdriver Monitoring
func NewTelemetrySink(logger lager.Logger, client MetricClient, projectID, subscriptionId, director string) telemetry.Sink {
	return &telemetrySink{
		logger:      logger,
		client:      client,
		projectPath: fmt.Sprintf("projects/%s", projectID),
		labels:      map[string]string{"subscription_id": subscriptionId, "director": director},
		startTime:   now(),
		resource:    detectMonitoredResource()}
}

func (ts *telemetrySink) Init(registeredSeries []*expvar.KeyValue) {
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   ts.projectPath,
		Filter: fmt.Sprintf(`metric.type = starts_with("stackdriver-nozzle")`),
	}

	descriptors, err := ts.client.ListMetricDescriptors(req)
	if err != nil {
		ts.logger.Error("telemetrySink.ListMetricDescriptors", err, lager.Data{"req": req})
	}

	registered := map[string]bool{}
	for _, descriptor := range descriptors {
		registered[descriptor.Name] = true
	}

	labels := []*labelpb.LabelDescriptor{}
	for name := range ts.labels {
		labels = append(labels, &labelpb.LabelDescriptor{Key: name, ValueType: labelpb.LabelDescriptor_STRING})
	}

	for _, series := range registeredSeries {
		name := ts.metricDescriptorName(series.Key)
		if registered[name] {
			continue
		}

		req := &monitoringpb.CreateMetricDescriptorRequest{
			Name: ts.projectPath,
			MetricDescriptor: &metricpb.MetricDescriptor{
				DisplayName: ts.metricDescriptorDisplayName(series.Key),
				Name:        name,
				Type:        ts.metricDescriptorType(series.Key),
				Labels:      labels,
				MetricKind:  metricpb.MetricDescriptor_CUMULATIVE,
				ValueType:   metricpb.MetricDescriptor_INT64,
				Description: "stackdriver-nozzle created custom metric.",
			},
		}
		err := ts.client.CreateMetricDescriptor(req)

		if err != nil {
			ts.logger.Error("telemetrySink.CreateMetricDescriptor", err, lager.Data{"req": req})
		}
	}
}

func (ts *telemetrySink) metricDescriptorDisplayName(key string) string {
	return fmt.Sprintf("stackdriver-nozzle/%s", key)
}

func (ts *telemetrySink) metricDescriptorName(key string) string {
	return fmt.Sprintf("%s/metricDescriptors/%s", ts.projectPath, ts.metricDescriptorType(key))
}

func (ts *telemetrySink) metricDescriptorType(key string) string {
	return fmt.Sprintf("custom.googleapis.com/stackdriver-nozzle/%s", key)
}

const maxTimeSeries = 200

func (ts *telemetrySink) newRequest() *monitoringpb.CreateTimeSeriesRequest {
	return &monitoringpb.CreateTimeSeriesRequest{
		Name: ts.projectPath,
	}
}

func (ts *telemetrySink) Report(report []*expvar.KeyValue) {
	req := ts.newRequest()

	timeInterval := &monitoringpb.TimeInterval{
		EndTime:   now(),
		StartTime: ts.startTime,
	}

	for _, data := range report {
		if val, ok := data.Value.(*expvar.Int); ok {
			req.TimeSeries = append(req.TimeSeries, &monitoringpb.TimeSeries{
				Metric: &metricpb.Metric{
					Type:   ts.metricDescriptorType(data.Key),
					Labels: ts.labels,
				},
				Points: []*monitoringpb.Point{{
					Interval: timeInterval,
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_Int64Value{Int64Value: val.Value()},
					},
				}},
				Resource: ts.resource,
			})
		}

		if len(req.TimeSeries) == maxTimeSeries {
			ts.client.Post(req)
			req = ts.newRequest()
		}
	}

	if len(req.TimeSeries) != 0 {
		ts.client.Post(req)
	}
}
