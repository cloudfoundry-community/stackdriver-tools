/*
 * Copyright 2017 Google Inc.
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
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	logId = "cf_logs"
)

var (
	logsCount *telemetry.Counter
)

func init() {
	logsCount = telemetry.NewCounter(telemetry.Nozzle, "logs.count")
}

type LogAdapter interface {
	PostLog(*messages.Log)
	Flush()
}

// NewLogAdapter returns a LogAdapter that can post to Stackdriver Logging.
func NewLogAdapter(projectID string, batchCount int, batchDuration time.Duration, inFlight int) (LogAdapter, <-chan error) {
	errs := make(chan error)
	loggingClient, err := logging.NewClient(context.Background(), projectID, option.WithUserAgent(version.UserAgent()))
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
		logging.ConcurrentWriteLimit(inFlight),
	)

	resource := &mrpb.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": projectID,
		},
	}

	return &logAdapter{
		sdLogger: sdLogger,
		resource: resource,
	}, errs
}

type logAdapter struct {
	sdLogger *logging.Logger
	resource *mrpb.MonitoredResource
}

// PostLog sends a single message to Stackdriver Logging
func (s *logAdapter) PostLog(log *messages.Log) {
	logsCount.Increment()
	entry := logging.Entry{
		Payload:  log.Payload,
		Labels:   log.Labels,
		Severity: log.Severity,
		Resource: s.resource,
	}
	s.sdLogger.Log(entry)
}

func (s *logAdapter) Flush() {
	s.sdLogger.Flush()
}
