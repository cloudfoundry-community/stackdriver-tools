package main

import (
	"fmt"

	"os"

	"cloud.google.com/go/monitoring/apiv3"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

func main() {
	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring"))
	if err != nil {
		panic(err)
	}

	projectID := os.Getenv("PROJECT_ID")

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
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

		fmt.Printf("Clearing %v\n", resp.Name)
		req := &monitoringpb.DeleteMetricDescriptorRequest{
			Name: resp.Name,
		}
		err = metricClient.DeleteMetricDescriptor(ctx, req)
		if err != nil {
			panic(err)
		}
	}
}
