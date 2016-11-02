package nozzle

import (
	"strings"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
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
	LogSink    Sink
	MetricSink Sink

	Heartbeater heartbeat.Heartbeater

	done chan struct{}
}

func (n *Nozzle) Start(fhClient firehose.Client) (errs chan error, fhErrs <-chan error) {
	n.Heartbeater.Start()
	n.done = make(chan struct{})

	errs = make(chan error)
	messages, fhErrs := fhClient.Connect()
	go func() {
		for {
			select {
			case envelope := <-messages:
				n.Heartbeater.Increment("nozzle.events")
				err := n.handleEvent(envelope)
				if err != nil {
					errs <- err
				}
			case <-n.done:
				return
			}
		}
	}()

	return errs, fhErrs
}

func (n *Nozzle) Stop() {
	n.Heartbeater.Stop()
	n.done <- struct{}{}
}

func (n *Nozzle) handleEvent(envelope *events.Envelope) error {
	var handler Sink
	if isLog(envelope) {
		handler = n.LogSink
	} else {
		handler = n.MetricSink
	}

	return handler.Receive(envelope)
}

func isLog(envelope *events.Envelope) bool {
	switch *envelope.EventType {
	case events.Envelope_ValueMetric, events.Envelope_ContainerMetric, events.Envelope_CounterEvent:
		return false
	default:
		return true
	}
}
