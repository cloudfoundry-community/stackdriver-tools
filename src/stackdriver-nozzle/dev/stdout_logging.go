package dev

import "github.com/cloudfoundry/sonde-go/events"

type StdOut struct{}

func (so *StdOut) HandleEvent(envelope *events.Envelope) error {
	println(envelope.String())
	return nil
}
