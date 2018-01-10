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

package nozzle

import (
	"errors"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/sonde-go/events"
)

var (
	blacklistedEvents *telemetry.Counter
	whitelistedEvents *telemetry.Counter
)

func init() {
	blacklistedEvents = telemetry.NewCounter(telemetry.Nozzle, "filter_sink.blacklisted_events")
	whitelistedEvents = telemetry.NewCounter(telemetry.Nozzle, "filter_sink.whitelisted_events")
}

type filter struct {
	enabled              map[events.Envelope_EventType]bool
	blacklist, whitelist *EventFilter
	destination          Sink
}

func NewFilterSink(eventNames []events.Envelope_EventType, blacklist, whitelist *EventFilter, destination Sink) (Sink, error) {
	if destination == nil {
		return nil, errors.New("missing destinationSink")
	}

	f := &filter{
		enabled:     make(map[events.Envelope_EventType]bool),
		blacklist:   blacklist,
		whitelist:   whitelist,
		destination: destination,
	}

	for _, eventType := range eventNames {
		f.enabled[eventType] = true
	}

	return f, nil
}

func (fs *filter) isBlacklisted(event *events.Envelope) bool {
	if fs.blacklist.Match(event) {
		if fs.whitelist.Match(event) {
			whitelistedEvents.Increment()
			return false
		}
		blacklistedEvents.Increment()
		return true
	}
	return false
}

func (fs *filter) Receive(event *events.Envelope) {
	if !fs.enabled[event.GetEventType()] || fs.isBlacklisted(event) {
		return
	}
	fs.destination.Receive(event)
}
