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
ClearMetricsDescriptors - delete custom MetricDescriptors from a Google Cloud Project

Setup:
- Setup application default credentials to a user with 'roles/monitoring.admin'
  `gcloud auth application-default login`
- Ensure your GOPATH is correct relative to the source repo.
  For example if you set your GOPATH to $HOME/GO
    export GOPATH=$HOME/go
  This file should be located at:
    $HOME/go/src/github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cmd/clear-metric-descriptors.go

Usage (from this directory):
go run ./clear-metric-descriptors.go --help
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/monitoring/apiv3"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

var (
	projectID string
	prefix    string
)

func init() {
	flag.StringVar(&projectID, "project-id", "", "The Google Cloud Project ID used for Stackdriver Monitoring, eg cf-prod-mon")
	flag.StringVar(&prefix, "prefix", "custom.googleapis.com/", "Prefix of metric.type for finding Metric Descriptors")
}

func main() {
	flag.Parse()

	if projectID == "" {
		log.Fatalf("error: project-id flag required, try runnig with --help")
	}

	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring"))
	if err != nil {
		panic(err)
	}

	log.Printf("discovering metric descriptors for %s", projectID)

	itr := metricClient.ListMetricDescriptors(ctx, &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
		Filter: fmt.Sprintf(`metric.type = starts_with("%s")`, prefix),
	})
	var names []string
	for resp, err := itr.Next(); err != iterator.Done; resp, err = itr.Next() {
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

	fmt.Printf("Delete %d metric descriptors from project?\n", len(names))
	fmt.Printf("This is irreversible and will result in data loss: (y/n) ")
	reader := bufio.NewReader(os.Stdin)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("could not read stdin: %v", err)
	}

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
