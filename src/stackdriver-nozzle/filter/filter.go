package filter

import (
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose"
)

type filter struct {
	dest    firehose.FirehoseHandler
	enabled map[events.Envelope_EventType]bool
}

func parseEventName(name string) (events.Envelope_EventType, error) {
	if eventId, ok := events.Envelope_EventType_value[name]; ok {
		return events.Envelope_EventType(eventId), nil
	}
	return events.Envelope_Error, fmt.Errorf("unknown event name: %s", name)
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

func DisplayValidEvents() {
	fmt.Printf("Valid event choices:")
	for _, name := range events.Envelope_EventType_name {
		fmt.Printf("- ", name)
	}
}
