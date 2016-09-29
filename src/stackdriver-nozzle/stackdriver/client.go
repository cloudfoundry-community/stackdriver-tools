package stackdriver

import (
	"time"

	"cloud.google.com/go/logging"
	"golang.org/x/net/context"
)

type Client interface {
	Post(payload interface{}, labels map[string]string)
}

type client struct {
	logger *logging.Logger
}

const (
	logId                = "cf_logs"
	DefaultBatchCount    = "10"
	DefaultBatchDuration = "1s"
)

// TODO error handling #131310523
func NewClient(projectID string, batchCount int, batchDuration time.Duration) Client {
	ctx := context.Background()

	loggingClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	loggingClient.OnError = func(err error) {
		panic(err)
	}

	logger := loggingClient.Logger(logId,
		logging.EntryCountThreshold(batchCount),
		logging.DelayThreshold(batchDuration))

	return &client{logger: logger}
}

func (s *client) Post(payload interface{}, _ map[string]string) {
	entry := logging.Entry{
		Payload: payload,
	}
	s.logger.Log(entry)
}
