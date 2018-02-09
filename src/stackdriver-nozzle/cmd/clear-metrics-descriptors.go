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

	"bufio"
	"log"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const projectIdEnv = "GCP_PROJECT_ID"

func main() {
	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring"))
	if err != nil {
		panic(err)
	}

	projectID := os.Getenv(projectIdEnv)
	if projectID == "" {
		log.Fatalf("error: environment variable %s is empty, set it to the project ID used for Stackdriver Monitoring", projectIdEnv)
	}

	log.Printf("discovering metric descriptors for %s", projectID)

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
		Filter: "metric.type = starts_with(\"custom.googleapis.com/\")",
	}
	it := metricClient.ListMetricDescriptors(ctx, req)
	var names []string
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("listing metric descriptors for %s: %v", projectID, err)
		}

		names = append(names, resp.Name)
		log.Printf("Found metric descriptor for deletion: %s\n", resp.Name)
	}

	if len(names) == 0 {
		log.Printf("no metric descriptors found for deletion")
		os.Exit(0)
	}

	fmt.Printf("Delete listed metric descriptors from project?\n")
	fmt.Printf("This is irreversible and will result in data loss: (y/n) ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')

	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		os.Exit(0)
	}

	for _, name := range names {
		log.Printf("Clearing: %s\n", name)
		req := &monitoringpb.DeleteMetricDescriptorRequest{
			Name: name,
		}
		err = metricClient.DeleteMetricDescriptor(ctx, req)
		if err != nil {
			log.Fatalf("deleting %s: %v", name, err)
		}
	}
}
