package stackdriver

import (
	"time"

	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type Client interface {
	PostLog(payload interface{}, labels map[string]string)
	PostMetric(name string, value float64) error
}

type client struct {
	ctx          context.Context
	logger       *logging.Logger
	metricClient *monitoring.MetricClient
	projectID    string
}

const (
	logId                = "cf_logs"
	DefaultBatchCount    = "10"
	DefaultBatchDuration = "1s"
)

// TODO error handling #131310523
func NewClient(projectID string, batchCount int, batchDuration time.Duration) Client {
	ctx := context.Background()

	logger, err := newLogger(ctx, projectID, batchCount, batchDuration)
	if err != nil {
		panic(err)
	}

	metricClient, err := monitoring.NewMetricClient(ctx, option.WithScopes("https://www.googleapis.com/auth/monitoring.write"))
	if err != nil {
		panic(err)
	}

	return &client{ctx: ctx, logger: logger, metricClient: metricClient, projectID: projectID}
}

func newLogger(ctx context.Context, projectID string, batchCount int, batchDuration time.Duration) (*logging.Logger, error) {
	loggingClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	loggingClient.OnError = func(err error) {
		panic(err)
	}

	logger := loggingClient.Logger(logId,
		logging.EntryCountThreshold(batchCount),
		logging.DelayThreshold(batchDuration),
	)
	return logger, nil
}

func (s *client) PostLog(payload interface{}, _ map[string]string) {
	entry := logging.Entry{
		Payload: payload,
	}
	s.logger.Log(entry)
}

func (s *client) PostMetric(name string, value float64) error {
	projectName := fmt.Sprintf("projects/%s", s.projectID)
	metricType := fmt.Sprintf("custom.googleapis.com/%s", name)

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: projectName,
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &google_api.Metric{
					Type:   metricType,
					Labels: map[string]string{},
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
	return s.metricClient.CreateTimeSeries(s.ctx, req)
}
