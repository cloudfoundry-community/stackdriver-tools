package stackdriver


import (
	"context"
	"fmt"
	"cloud.google.com/go/logging"
)

type Logger struct {
	client *logging.Client
}

func (lg *Logger) Publish(message string) {
	lg.client.Logger("stackdriver-spinner-logs").Log(logging.Entry{Payload: message})


	if err := lg.client.Close() ; err != nil {
		fmt.Errorf("Failed to close client: %v", err)
	}
}

func NewLogger(projectId string) (*Logger, error) {
	client, err := logging.NewClient(context.Background(), projectId)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &Logger{client}, nil
}
