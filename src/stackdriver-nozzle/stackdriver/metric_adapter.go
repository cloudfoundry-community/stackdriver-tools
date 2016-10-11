package stackdriver

import (
	"path"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	EventTime time.Time
}

type MetricAdapter interface {
	PostMetrics([]Metric) error
}

type metricAdapter struct {
	projectID string
	client    MetricClient
}

func NewMetricAdapter(projectID string, client MetricClient) MetricAdapter {
	return &metricAdapter{
		projectID: projectID,
		client:    client,
	}
}

func (ma *metricAdapter) PostMetrics(metrics []Metric) error {
	projectName := path.Join("projects", ma.projectID)
	var timeSerieses []*monitoringpb.TimeSeries

	for _, metric := range metrics {
		eventTime := metric.EventTime
		timeStamp := timestamp.Timestamp{
			Seconds: eventTime.Unix(),
			Nanos:   int32(eventTime.Nanosecond()),
		}

		metricType := path.Join("custom.googleapis.com", metric.Name)
		timeSeries := monitoringpb.TimeSeries{
			Metric: &google_api.Metric{
				Type:   metricType,
				Labels: metric.Labels,
			},
			Points: []*monitoringpb.Point{
				{
					Interval: &monitoringpb.TimeInterval{
						EndTime:   &timeStamp,
						StartTime: &timeStamp,
					},
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_DoubleValue{
							DoubleValue: metric.Value,
						},
					},
				},
			},
		}
		timeSerieses = append(timeSerieses, &timeSeries)
	}

	request := &monitoringpb.CreateTimeSeriesRequest{
		Name:       projectName,
		TimeSeries: timeSerieses,
	}

	return ma.client.Post(request)
}
