package mocks

import (
	"sync"

	"github.com/cloudfoundry/sonde-go/events"
)

type Sink struct {
	HandledEnvelopes []events.Envelope
	Error            error
	mutex            sync.Mutex
}

func (s *Sink) Receive(envelope *events.Envelope) error {
	s.mutex.Lock()
	s.HandledEnvelopes = append(s.HandledEnvelopes, *envelope)
	s.mutex.Unlock()

	return s.Error
}

func (s *Sink) LastEnvelope() *events.Envelope {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.HandledEnvelopes) == 0 {
		return nil
	}

	return &s.HandledEnvelopes[len(s.HandledEnvelopes)-1]
}
