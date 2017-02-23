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
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const (
	logId = "cf_logs"
)

type LogAdapter interface {
	PostLog(*Log)
	Flush()
}

type Log struct {
	Payload  interface{}
	Labels   map[string]string
	Severity logging.Severity
}

func NewLogAdapter(projectID string, batchCount int, batchDuration time.Duration, heartbeater Heartbeater) (LogAdapter, <-chan error) {
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
	)

	return &logAdapter{
		sdLogger:    sdLogger,
		heartBeater: heartbeater,
	}, errs
}

type logAdapter struct {
	sdLogger    *logging.Logger
	heartBeater Heartbeater
}

func (s *logAdapter) PostLog(log *Log) {
	s.heartBeater.Increment("logs.count")
	entry := logging.Entry{
		Payload:  log.Payload,
		Labels:   log.Labels,
		Severity: log.Severity,
	}
	s.sdLogger.Log(entry)
}

func (s *logAdapter) Flush() {
	s.sdLogger.Flush()
}
