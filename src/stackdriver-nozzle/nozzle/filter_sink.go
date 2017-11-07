package nozzle

import (
	"errors"

	"github.com/cloudfoundry/sonde-go/events"
)

type filter struct {
	enabled     map[events.Envelope_EventType]bool
	destination Sink
}

func NewFilterSink(eventNames []events.Envelope_EventType, destination Sink) (Sink, error) {
	if destination == nil {
		return nil, errors.New("missing destinationSink")
	}

	f := &filter{enabled: make(map[events.Envelope_EventType]bool), destination: destination}

	for _, eventType := range eventNames {
		f.enabled[eventType] = true
	}

	return f, nil
}

func (sf *filter) Receive(event *events.Envelope) {
	if sf.enabled[event.GetEventType()] {
		sf.destination.Receive(event)
	}
}
