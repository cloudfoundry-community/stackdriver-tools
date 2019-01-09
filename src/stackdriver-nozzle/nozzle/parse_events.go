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

package nozzle

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/sonde-go/events"
)

func ParseEvents(names []string) ([]events.Envelope_EventType, error) {
	var parsedEvents []events.Envelope_EventType

	for _, name := range names {
		if name == "" {
			continue
		}

		event, err := parseEventName(name)
		if err != nil {
			return nil, err
		}
		parsedEvents = append(parsedEvents, event)
	}

	return parsedEvents, nil
}

func parseEventName(name string) (events.Envelope_EventType, error) {
	if eventID, ok := events.Envelope_EventType_value[name]; ok {
		return events.Envelope_EventType(eventID), nil
	}
	return events.Envelope_Error, &invalidEvent{name: name}
}

type invalidEvent struct {
	name string
}

func (ie *invalidEvent) Error() string {
	var eventNames []string
	for _, name := range events.Envelope_EventType_name {
		eventNames = append(eventNames, name)
	}
	validEvents := strings.Join(eventNames, ",")

	return fmt.Sprintf("invalid event '%s'; valid events: %s", ie.name, validEvents)
}
