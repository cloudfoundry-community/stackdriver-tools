/*
 * Copyright 2019 Google Inc.
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

package mocks

import (
	"sync"

	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/monitoring/v3"
)

type MockClient struct {
	Mutex          sync.Mutex
	MetricReqs     []*monitoring.CreateTimeSeriesRequest
	TimeSeries     []*monitoring.TimeSeries
	DescriptorReqs []*monitoring.CreateMetricDescriptorRequest
	ListErr        error

	CreateMetricDescriptorFn func(req *monitoring.CreateMetricDescriptorRequest) error
	ListMetricDescriptorFn   func(request *monitoring.ListMetricDescriptorsRequest) ([]*metric.MetricDescriptor, error)
	PostFn                   func(req *monitoring.CreateTimeSeriesRequest) error
}

func (mc *MockClient) Post(req *monitoring.CreateTimeSeriesRequest) error {
	if mc.PostFn != nil {
		return mc.PostFn(req)
	}

	mc.Mutex.Lock()
	mc.MetricReqs = append(mc.MetricReqs, req)
	mc.TimeSeries = append(mc.TimeSeries, req.TimeSeries...)
	mc.Mutex.Unlock()

	return nil
}

func (mc *MockClient) CreateMetricDescriptor(request *monitoring.CreateMetricDescriptorRequest) error {
	if mc.CreateMetricDescriptorFn != nil {
		return mc.CreateMetricDescriptorFn(request)
	}

	mc.Mutex.Lock()
	mc.DescriptorReqs = append(mc.DescriptorReqs, request)
	mc.Mutex.Unlock()

	return nil
}

func (mc *MockClient) ListMetricDescriptors(request *monitoring.ListMetricDescriptorsRequest) ([]*metric.MetricDescriptor, error) {
	if mc.ListMetricDescriptorFn != nil {
		return mc.ListMetricDescriptorFn(request)
	}

	if mc.ListErr != nil {
		return nil, mc.ListErr
	}
	return []*metric.MetricDescriptor{
		{Name: "anExistingMetric"},
	}, nil
}
