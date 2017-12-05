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

/*
ClearMetricsDescriptors - delete _all_ custom MetricDescriptors from a Google Cloud Project

Setup:
- Export environment variable GCP_PROJECT_ID=<GCP Project for Stackdriver Monitoring>
- Setup application default credentials to a user with 'roles/monitoring.admin'
  `gcloud auth application-default login`
- Ensure your GOPATH is correct relative to the source repo.
  For example if you set your GOPATH to $HOME/GO
    export GOPATH=$HOME/go
  This file should be located at:
    $HOME/go/src/github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cmd/clear-metric-descriptors.go

Usage (from this directory):
go run ./clear-metric-descriptors.go
*/
package main

import (
	"fmt"
	"os"

	monitoring "cloud.google.com/go/monitoring/apiv3"

	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

func main() {
	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring"))
	if err != nil {
		panic(err)
	}

	projectID := os.Getenv("GCP_PROJECT_ID")

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
		Filter: "metric.type = starts_with(\"custom.googleapis.com/\")",
	}
	it := metricClient.ListMetricDescriptors(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
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
