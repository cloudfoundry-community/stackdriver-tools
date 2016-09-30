package dev

import (
	"cloud.google.com/go/monitoring/apiv3"
	"github.com/evandbrown/gcp-tools-release/src/firehose-to-fluentd/Godeps/_workspace/src/golang.org/x/net/context"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

func ClearMetricDescriptors() {
	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring"))
	if err != nil {
		panic(err)
	}

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   "projects/evandbrown17",
		Filter: "metric.type = starts_with(\"custom.googleapis.com/\")",
	}
	it := metricClient.ListMetricDescriptors(ctx, req)
	for {
		resp, err := it.Next()
		if err == monitoring.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
			panic(err)
		}

		req := &monitoringpb.DeleteMetricDescriptorRequest{
			Name: resp.Name,
		}
		err = metricClient.DeleteMetricDescriptor(ctx, req)
		if err != nil {
			panic(err)
		}
	}
}
