package mocks

import "github.com/cloudfoundry/sonde-go/events"

type Sink struct {
	HandledEnvelopes []events.Envelope
	Error            error
}

func (h *Sink) Receive(envelope *events.Envelope) error {
	h.HandledEnvelopes = append(h.HandledEnvelopes, *envelope)
	return h.Error
}
