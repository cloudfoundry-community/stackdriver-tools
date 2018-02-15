/*
 * Copyright 2019 Google Inc.
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

package stackdriver

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
)

type Logger struct {
	client     *logging.Client
	foundation string
}

type Message struct {
	GUID             string  `json:"guid"`
	NumberSent       int     `json:"number_sent"`
	NumberFound      int     `json:"number_found"`
	BurstIntervalSec int     `json:"burst_interval_sec"`
	LossPercentage   float64 `json:"loss_percentage"`
}

func (lg *Logger) Publish(message Message) {
	lg.client.Logger("stackdriver-spinner-logs").Log(logging.Entry{Payload: message, Labels: map[string]string{"foundation": lg.foundation}})

	if err := lg.client.Close(); err != nil {
		fmt.Errorf("Failed to close client: %v", err)
	}
}

func NewLogger(projectID, foundation string) (*Logger, error) {
	client, err := logging.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &Logger{client, foundation}, nil
}
