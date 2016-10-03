package main

import (
	"errors"
	"fmt"
	"path"
	"time"

	"os"

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

		This piece of code is an attempt to repro the error without any firehose/nozzle interaction
	*/
	ctx := context.Background()
	metricClient, _ := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.write"))

	t := time.NewTicker(100 * time.Millisecond)
	errCount := 0
	for _ = range t.C {
		println(fmt.Sprintf("tick: %v, %v", errCount, time.Now().Second()))
		err := postMetric(metricClient, ctx, "andres_test", float64(time.Now().Second()), map[string]string{})
		if err != nil {
			errCount += 1
			fmt.Printf("A wild error #%v appeared: %v\n", errCount, err)
		}
		if errCount > 10 {
			panic(errors.New("too many errors"))
		}
	}
}

func postMetric(m *monitoring.MetricClient, ctx context.Context, name string, value float64, labels map[string]string) error {
	projectID := os.Getenv("PROJECT_ID")
	projectName := fmt.Sprintf("projects/%s", projectID)
	metricType := path.Join("custom.googleapis.com", name)

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: projectName,
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &google_api.Metric{
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
