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

package filter

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry/sonde-go/events"
)

func New(firehose cloudfoundry.Firehose, eventNames []string, heartbeater heartbeat.Heartbeater) (cloudfoundry.Firehose, error) {
	f := filter{firehose: firehose, enabled: make(map[events.Envelope_EventType]bool), heartbeater: heartbeater}

	for _, eventName := range eventNames {
		eventType, err := parseEventName(eventName)

		if err != nil {
			return nil, err
		}

		f.enabled[eventType] = true
	}

	return &f, nil
}

type filter struct {
	firehose    cloudfoundry.Firehose
	enabled     map[events.Envelope_EventType]bool
	heartbeater heartbeat.Heartbeater
}

func (f *filter) Connect() (<-chan *events.Envelope, <-chan error) {
	filteredMessages := make(chan *events.Envelope)
	messages, errs := f.firehose.Connect()

	go func() {
		for envelope := range messages {
			f.heartbeater.Increment("filter.events")
			if f.enabled[envelope.GetEventType()] {
				filteredMessages <- envelope
			}
		}
	}()

	return filteredMessages, errs
}

func parseEventName(name string) (events.Envelope_EventType, error) {
	if eventId, ok := events.Envelope_EventType_value[name]; ok {
		return events.Envelope_EventType(eventId), nil
	}
	return events.Envelope_Error, &invalidEvent{name: name}
}

type invalidEvent struct {
	name string
}

func (ie *invalidEvent) Error() string {
	eventNames := []string{}
	for _, name := range events.Envelope_EventType_name {
		eventNames = append(eventNames, name)
	}
	validEvents := strings.Join(eventNames, ",")

	return fmt.Sprintf("invalid event '%s'; valid events: %s", ie.name, validEvents)
}
