package filter

import (
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
	"stackdriver-nozzle/firehose"
	"strings"
)

type filter struct {
	dest    firehose.FirehoseHandler
	enabled map[events.Envelope_EventType]bool
}

type invalidEvent struct {
	name string
}

func parseEventName(name string) (events.Envelope_EventType, error) {
	if eventId, ok := events.Envelope_EventType_value[name]; ok {
		return events.Envelope_EventType(eventId), nil
	}
	return events.Envelope_Error, &invalidEvent{name: name}
}

func (ie *invalidEvent) Error() string {
	eventNames := []string{}
	for _, name := range events.Envelope_EventType_name {
		eventNames = append(eventNames, name)
	}
	validEvents := strings.Join(eventNames, ", ")

	return fmt.Sprintf("invalid event '%s'; valid events: %s", ie.name, validEvents)
}

func New(dest firehose.FirehoseHandler, eventNames []string) (firehose.FirehoseHandler, error) {
	f := filter{dest: dest, enabled: make(map[events.Envelope_EventType]bool)}

	for _, eventName := range eventNames {
		eventType, err := parseEventName(eventName)

		if err != nil {
			return nil, err
		}

		f.enabled[eventType] = true
	}

	return &f, nil
}

func (f *filter) HandleEvent(envelope *events.Envelope) error {
	if !f.enabled[envelope.GetEventType()] {
		return nil
	}
	return f.dest.HandleEvent(envelope)
}
