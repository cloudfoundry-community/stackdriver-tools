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

	"cloud.google.com/go/monitoring/apiv3"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

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
	return m.sdMetricClient.CreateTimeSeries(m.ctx, request)
}

func (m *metricClient) CreateMetricDescriptor(request *monitoringpb.CreateMetricDescriptorRequest) error {
	_, err := m.sdMetricClient.CreateMetricDescriptor(m.ctx, request)
	return err
}

func (m *metricClient) ListMetricDescriptors(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
	it := m.sdMetricClient.ListMetricDescriptors(m.ctx, request)

	descriptors := []*metricpb.MetricDescriptor{}
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
