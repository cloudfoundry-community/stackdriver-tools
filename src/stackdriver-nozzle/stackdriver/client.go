package stackdriver

import (
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry/lager"
	"golang.org/x/net/context"
)

type Client interface {
	PostLog(payload interface{}, labels map[string]string)
}

type client struct {
	ctx       context.Context
	sdLogger  *logging.Logger
	projectID string
	logger    lager.Logger
}

const (
	logId                = "cf_logs"
	DefaultBatchCount    = "10"
	DefaultBatchDuration = "1s"
)

// TODO error handling #131310523
func NewClient(projectID string, batchCount int, batchDuration time.Duration, logger lager.Logger) Client {
	ctx := context.Background()

	sdLogger := newLogger(ctx, projectID, batchCount, batchDuration, logger)

	return &client{
		ctx:       ctx,
		sdLogger:  sdLogger,
		projectID: projectID,
		logger:    logger,
	}
}

func newLogger(ctx context.Context, projectID string, batchCount int, batchDuration time.Duration, logger lager.Logger) *logging.Logger {
	loggingClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		logger.Fatal("stackdriverClient", err)
	}

	loggingClient.OnError = func(err error) {
		logger.Fatal("stackdriverClientOnError", err)
	}

	return loggingClient.Logger(logId,
		logging.EntryCountThreshold(batchCount),
		logging.DelayThreshold(batchDuration),
	)
}

func (s *client) PostLog(payload interface{}, labels map[string]string) {
	entry := logging.Entry{
		Payload: payload,
		Labels:  labels,
	}
	s.sdLogger.Log(entry)
}
