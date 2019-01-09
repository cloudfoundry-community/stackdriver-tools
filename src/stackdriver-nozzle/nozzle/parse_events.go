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
