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
	"strings"

	"cloud.google.com/go/monitoring/apiv3"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

var (
	timeSeriesReqs *telemetry.Counter
	timeSeriesErrs *telemetry.CounterMap

	timeSeriesErrOutOfOrder *telemetry.Counter
	timeSeriesErrUnknown    *telemetry.Counter

	descriptorReqs *telemetry.Counter
	descriptorErrs *telemetry.Counter
)

func init() {
	timeSeriesReqs = telemetry.NewCounter(telemetry.Nozzle, "metrics.timeseries.requests")
	timeSeriesErrs = telemetry.NewCounterMap(telemetry.Nozzle, "metrics.timeseries.errors", "error_type")

	timeSeriesErrOutOfOrder = timeSeriesErrs.MustCounter("out_of_order")
	timeSeriesErrUnknown = timeSeriesErrs.MustCounter("unknown")

	descriptorReqs = telemetry.NewCounter(telemetry.Nozzle, "metrics.descriptor.requests")
	descriptorErrs = telemetry.NewCounter(telemetry.Nozzle, "metrics.descriptor.errors")
}

type MetricClient interface {
	Post(*monitoringpb.CreateTimeSeriesRequest) error
	CreateMetricDescriptor(*monitoringpb.CreateMetricDescriptorRequest) error
	ListMetricDescriptors(*monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error)
}

func NewMetricClient() (MetricClient, error) {
	ctx := context.Background()
	sdMetricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.write"), option.WithUserAgent(version.UserAgent()))
	if err != nil {
		return nil, err
	}

	return &metricClient{
		sdMetricClient: sdMetricClient,
		ctx:            ctx,
	}, nil
}

type metricClient struct {
	sdMetricClient *monitoring.MetricClient
	ctx            context.Context
}

func (m *metricClient) Post(request *monitoringpb.CreateTimeSeriesRequest) error {
	timeSeriesReqs.Increment()
	err := m.sdMetricClient.CreateTimeSeries(m.ctx, request)
	if err != nil {
		if strings.Contains(err.Error(), `Points must be written in order`) {
			// This is an expected error once there is more than a single nozzle writing to Stackdriver.
			// If one nozzle writes a metric occurring at time T2 and this one tries to write at T1 (where T2 later than T1)
			// we will receive this error.
			timeSeriesErrOutOfOrder.Increment()
			return nil // absorb error
		}
		timeSeriesErrUnknown.Increment()
	}
	return err
}

func (m *metricClient) CreateMetricDescriptor(request *monitoringpb.CreateMetricDescriptorRequest) error {
	descriptorReqs.Increment()
	_, err := m.sdMetricClient.CreateMetricDescriptor(m.ctx, request)
	if err != nil {
		descriptorErrs.Increment()
	}
	return err
}

func (m *metricClient) ListMetricDescriptors(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
	it := m.sdMetricClient.ListMetricDescriptors(m.ctx, request)

	var descriptors []*metricpb.MetricDescriptor
	for {
		metricDescriptor, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		descriptors = append(descriptors, metricDescriptor)
	}

	return descriptors, nil

}
