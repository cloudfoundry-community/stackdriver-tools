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
	LogHandler    LogHandler
	MetricHandler MetricHandler
	Serializer    serializer.Serializer
	Heartbeater   heartbeat.Heartbeater
}

func (n *Nozzle) HandleEvent(envelope *events.Envelope) error {
	if n.Serializer.IsLog(envelope) {
		n.Heartbeater.AddCounter()
		n.LogHandler.HandleEnvelope(envelope)
		return nil
	} else {
		n.Heartbeater.AddCounter()
		return n.MetricHandler.HandleEnvelope(envelope)
	}
}
