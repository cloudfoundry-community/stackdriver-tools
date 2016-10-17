package stackdriver

import (
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/version"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const (
	logId = "cf_logs"
)

type LogAdapter interface {
	PostLog(*Log)
}

type Log struct {
	Payload interface{}
	Labels  map[string]string
}

func NewLogAdapter(projectID string, batchCount int, batchDuration time.Duration, onError func(error)) (LogAdapter, error) {
	loggingClient, err := logging.NewClient(context.Background(), projectID, option.WithUserAgent(version.UserAgent))
	if err != nil {
		return nil, err
	}

	loggingClient.OnError = onError

	sdLogger := loggingClient.Logger(logId,
		logging.EntryCountThreshold(batchCount),
		logging.DelayThreshold(batchDuration),
	)

	return &logClient{
		sdLogger: sdLogger,
	}, nil
}

type logClient struct {
	sdLogger *logging.Logger
}

func (s *logClient) PostLog(log *Log) {
	entry := logging.Entry{
		Payload: log.Payload,
		Labels:  log.Labels,
	}
	s.sdLogger.Log(entry)
}
