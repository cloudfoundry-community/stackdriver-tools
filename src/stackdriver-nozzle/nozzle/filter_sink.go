package nozzle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloudfoundry/sonde-go/events"
)

type filter struct {
	enabled     map[events.Envelope_EventType]bool
	destination Sink
}

func NewFilterSink(eventNames []string, destination Sink) (Sink, error) {
	if destination == nil {
		return nil, errors.New("missing destinationSink")
	}

	f := &filter{enabled: make(map[events.Envelope_EventType]bool), destination: destination}

	for _, eventName := range eventNames {
		eventType, err := parseEventName(eventName)

		if err != nil {
			return nil, err
		}

		f.enabled[eventType] = true
	}

	return f, nil
}

func (sf *filter) Receive(event *events.Envelope) error {
	if sf.enabled[event.GetEventType()] {
		return sf.destination.Receive(event)
	}
	return nil
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
