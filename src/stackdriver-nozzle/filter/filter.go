package filter

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
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
