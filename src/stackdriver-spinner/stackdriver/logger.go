package stackdriver


import (
	"context"
	"fmt"
	"cloud.google.com/go/logging"
)

type Logger struct {
	client *logging.Client
}

type Message struct {
	GUID             string  `json:"guid"`
	NumberSent       int     `json:"number_sent"`
	NumberFound      int     `json:"number_found"`
	BurstIntervalSec int     `json:"burst_interval_sec"`
	LossPercentage   float64 `json:"loss_percentage"`
}

func (lg *Logger) Publish(message Message) {
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
