package mocks

import (
	"sync"

	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type MockClient struct {
	Mutex          sync.Mutex
	MetricReqs     []*monitoringpb.CreateTimeSeriesRequest
	TimeSeries     []*monitoringpb.TimeSeries
	DescriptorReqs []*monitoringpb.CreateMetricDescriptorRequest
	ListErr        error

	CreateMetricDescriptorFn func(request *monitoringpb.CreateMetricDescriptorRequest) error
	PostFn                   func(req *monitoringpb.CreateTimeSeriesRequest) error
}

func (mc *MockClient) Post(req *monitoringpb.CreateTimeSeriesRequest) error {
	if mc.PostFn != nil {
		return mc.PostFn(req)
	}

	mc.Mutex.Lock()
	mc.MetricReqs = append(mc.MetricReqs, req)
	mc.TimeSeries = append(mc.TimeSeries, req.TimeSeries...)
	mc.Mutex.Unlock()

	return nil
}

func (mc *MockClient) CreateMetricDescriptor(request *monitoringpb.CreateMetricDescriptorRequest) error {
	if mc.CreateMetricDescriptorFn != nil {
		return mc.CreateMetricDescriptorFn(request)
	}

	mc.Mutex.Lock()
	mc.DescriptorReqs = append(mc.DescriptorReqs, request)
	mc.Mutex.Unlock()

	return nil
}

func (mc *MockClient) ListMetricDescriptors(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
	if mc.ListErr != nil {
		return nil, mc.ListErr
	}
	return []*metricpb.MetricDescriptor{
		{Name: "anExistingMetric"},
	}, nil
}
