package nozzle

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
)

type Nozzle struct {
	StackdriverClient stackdriver.Client
}

func (n *Nozzle) HandleEvent(eventsEnvelope *events.Envelope) error {
	envelope := Envelope{eventsEnvelope}

	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric:
		valueMetric := envelope.GetValueMetric()
		name := valueMetric.GetName()
		value := valueMetric.GetValue()

		err := n.StackdriverClient.PostMetric(name, value, envelope.Labels())
		return err
	default:
		n.StackdriverClient.PostLog(envelope, envelope.Labels())
		return nil
	}
}
