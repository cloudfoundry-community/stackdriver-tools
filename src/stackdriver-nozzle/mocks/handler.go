package mocks

import "github.com/cloudfoundry/sonde-go/events"

type Handler struct {
	HandledEnvelopes []events.Envelope
	Error            error
}

func (h *Handler) HandleEnvelope(envelope *events.Envelope) error {
	h.HandledEnvelopes = append(h.HandledEnvelopes, *envelope)
	return h.Error
}
