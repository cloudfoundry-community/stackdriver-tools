package stackdriver

import (
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
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

func NewLogAdapter(projectID string, batchCount int, batchDuration time.Duration, heartbeater heartbeat.Heartbeater) (LogAdapter, <-chan error) {
	errs := make(chan error)
	loggingClient, err := logging.NewClient(context.Background(), projectID, option.WithUserAgent(version.UserAgent))
	if err != nil {
		go func() { errs <- err }()
		return nil, errs
	}

	loggingClient.OnError = func(err error) {
		errs <- err
	}

	sdLogger := loggingClient.Logger(logId,
		logging.EntryCountThreshold(batchCount),
		logging.DelayThreshold(batchDuration),
	)

	return &logAdapter{
		sdLogger:    sdLogger,
		heartBeater: heartbeater,
	}, errs
}

type logAdapter struct {
	sdLogger    *logging.Logger
	heartBeater heartbeat.Heartbeater
}

func (s *logAdapter) PostLog(log *Log) {
	s.heartBeater.Increment("logs.count")
	entry := logging.Entry{
		Payload: log.Payload,
		Labels:  log.Labels,
	}
	s.sdLogger.Log(entry)
}
