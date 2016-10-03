package filter

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose"
)

type filter struct {
	dest    firehose.FirehoseHandler
	enabled map[events.Envelope_EventType]bool
}

func parseEventName(string) (events.Envelope_EventType, error) {
	panic("NYI")
}

func New(dest firehose.FirehoseHandler, events []string) (firehose.FirehoseHandler, error) {
	f := filter{}
	f.dest = dest

	for _, eventName := range events {
		eventType, err := parseEventName(eventName)

		if err != nil {
			return nil, err
		}

		f.enabled[eventType] = true
	}

	return f, nil
}

func (f filter) HandleEvent(envelope *events.Envelope) error {
	if f.enabled[envelope.GetEventType()] {
		return f.dest.HandleEvent(envelope)
	}
	return nil
}
