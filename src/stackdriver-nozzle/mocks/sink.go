package mocks

import "github.com/cloudfoundry/sonde-go/events"

type Sink struct {
	HandledEnvelopes []events.Envelope
	Error            error
}

func (s *Sink) Receive(envelope *events.Envelope) error {
	s.HandledEnvelopes = append(s.HandledEnvelopes, *envelope)
	return s.Error
}

func (s *Sink) LastEnvelope() *events.Envelope {
	if len(s.HandledEnvelopes) == 0 {
		return nil
	}

	return &s.HandledEnvelopes[len(s.HandledEnvelopes)-1]
}
