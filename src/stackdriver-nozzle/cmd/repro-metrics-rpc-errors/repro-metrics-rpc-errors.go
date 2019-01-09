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

package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

func main() {
	/*
		The error we are seeing is this:
		rpc error: code = 13 desc = stream terminated by RST_STREAM with error code: 2

		This piece of code is an attempt to reproduce the error without any firehose/nozzle interaction
	*/
	ctx := context.Background()
	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.write"))
	if err != nil {
		panic(fmt.Sprintf("Error creating metric client: %v", err))
	}
	t := time.NewTicker(100 * time.Millisecond)
	errCount := 0
	for range t.C {
		println(fmt.Sprintf("tick: %v, %v", errCount, time.Now().Second()))
		err := postMetric(ctx, metricClient, "andres_test", float64(time.Now().Second()), map[string]string{})
		if err != nil {
			errCount++
			fmt.Printf("A wild error #%v appeared: %v\n", errCount, err)
		}
		if errCount > 10 {
			panic(errors.New("too many errors"))
		}
	}
}

func postMetric(ctx context.Context, m *monitoring.MetricClient, name string, value float64, labels map[string]string) error {
	projectID := os.Getenv("GCP_PROJECT_ID")
	projectName := fmt.Sprintf("projects/%s", projectID)
	metricType := path.Join("custom.googleapis.com", name)

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: projectName,
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &metric.Metric{
					Type:   metricType,
					Labels: labels,
				},
				Points: []*monitoringpb.Point{
					{
						Interval: &monitoringpb.TimeInterval{
							EndTime: &timestamp.Timestamp{
								Seconds: time.Now().Unix(),
							},
							StartTime: &timestamp.Timestamp{
								Seconds: time.Now().Unix(),
							},
						},
						Value: &monitoringpb.TypedValue{
							Value: &monitoringpb.TypedValue_DoubleValue{
								DoubleValue: value,
							},
						},
					},
				},
			},
		},
	}
	err := m.CreateTimeSeries(ctx, req)
	return err
}
