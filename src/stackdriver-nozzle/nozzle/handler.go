package nozzle

import "github.com/cloudfoundry/sonde-go/events"

type Handler interface {
	HandleEnvelope(*events.Envelope) error
}
