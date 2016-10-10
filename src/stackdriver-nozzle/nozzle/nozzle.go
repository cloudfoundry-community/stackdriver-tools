package nozzle

import (
	"strings"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/serializer"
	"github.com/cloudfoundry/sonde-go/events"
)

type PostMetricError struct {
	Errors []error
}

func (e *PostMetricError) Error() string {
	errors := []string{}
	for _, err := range e.Errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, "\n")
}

type Nozzle struct {
	LogHandler    Handler
	MetricHandler Handler
	Serializer    serializer.Serializer
	Heartbeater   heartbeat.Heartbeater
}

func (n *Nozzle) HandleEvent(envelope *events.Envelope) error {
	var handler Handler
	if n.Serializer.IsLog(envelope) {
		handler = n.LogHandler
	} else {
		handler = n.MetricHandler
	}

	n.Heartbeater.AddCounter()
	return handler.HandleEnvelope(envelope)
}
