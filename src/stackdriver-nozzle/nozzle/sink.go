package nozzle

import "github.com/cloudfoundry/sonde-go/events"

type Sink interface {
	Receive(*events.Envelope) error
}
