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
