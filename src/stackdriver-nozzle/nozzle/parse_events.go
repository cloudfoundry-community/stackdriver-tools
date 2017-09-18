package nozzle

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/sonde-go/events"
)

func ParseEvents(names []string) ([]events.Envelope_EventType, error) {
	events := []events.Envelope_EventType{}

	for _, name := range names {
		event, err := parseEventName(name)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
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
