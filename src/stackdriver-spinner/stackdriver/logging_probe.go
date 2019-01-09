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
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

type LoggingProbe struct {
	client *logadmin.Client
}

func (lp *LoggingProbe) Find(start time.Time, needle string, count int) (int, error) {
	timeBytes, err := start.MarshalText()
	if err != nil {
		return 0, fmt.Errorf("problem marshaling text: %v", err)
	}

	it := lp.client.Entries(context.Background(), logadmin.Filter(fmt.Sprintf("jsonPayload.eventType=\"LogMessage\" timestamp>=\"%s\" jsonPayload.message:\"%s\"", timeBytes, needle)))

	var entries []*logging.Entry

	for {
		var err error
		pageToken := ""
		pageToken, err = iterator.NewPager(it, 1000, pageToken).NextPage(&entries)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return 0, fmt.Errorf("problem getting the next page: %v", err)
		}

		if pageToken == "" {
			break
		}
	}

	return len(entries), nil
}

func NewLoggingProbe(projectId string) (*LoggingProbe, error) {
	client, err := logadmin.NewClient(context.Background(), projectId)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &LoggingProbe{client}, nil
}
