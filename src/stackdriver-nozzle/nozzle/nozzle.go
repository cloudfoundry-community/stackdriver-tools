package nozzle

import (
	"strings"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
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
	LogAdapter    stackdriver.LogAdapter
	MetricAdapter stackdriver.MetricAdapter
	Serializer    serializer.Serializer
	Heartbeater   heartbeat.Heartbeater
}

func (n *Nozzle) HandleEvent(envelope *events.Envelope) error {
	if n.Serializer.IsLog(envelope) {
		log := n.Serializer.GetLog(envelope)
		n.Heartbeater.AddCounter()
		n.LogAdapter.PostLog(log)
		return nil
	} else {
		metrics, err := n.Serializer.GetMetrics(envelope)
		if err != nil {
			return err
		}
		n.Heartbeater.AddCounter()
		return n.MetricAdapter.PostMetrics(metrics)
	}
}
