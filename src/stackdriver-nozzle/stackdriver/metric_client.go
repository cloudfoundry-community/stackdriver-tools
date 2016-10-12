package stackdriver

import (
	"context"

	"cloud.google.com/go/monitoring/apiv3"
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
	sdMetricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.write"))
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
		if err == monitoring.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		descriptors = append(descriptors, metricDescriptor)
	}

	return descriptors, nil

}
