package nozzle

import (
	"strings"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/serializer"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
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
	StackdriverClient stackdriver.Client
	Serializer        serializer.Serializer
	MetricAdapter     stackdriver.MetricAdapter
}

func (n *Nozzle) HandleEvent(envelope *events.Envelope) error {
	if n.Serializer.IsLog(envelope) {
		log := n.Serializer.GetLog(envelope)
		n.StackdriverClient.PostLog(log.Payload, log.Labels)
		return nil
	} else {
		metrics, err := n.Serializer.GetMetrics(envelope)
		if err != nil {
			return err
		}
		return n.MetricAdapter.PostMetrics(metrics)
	}
}
