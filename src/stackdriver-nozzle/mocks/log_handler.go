package mocks

import "github.com/cloudfoundry/sonde-go/events"

type LogHandler struct {
	HandledEnvelopes []events.Envelope
}

func (lh *LogHandler) HandleEnvelope(envelope *events.Envelope) {
	lh.HandledEnvelopes = append(lh.HandledEnvelopes, *envelope)
}
