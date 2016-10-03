package nozzle

import (
	"github.com/cloudfoundry/sonde-go/events"
)

type Envelope struct {
	*events.Envelope
}

func (e *Envelope) Labels() map[string]string {
	labels := map[string]string{}

	if e.Origin != nil {
		labels["origin"] = e.GetOrigin()
	}

	if e.EventType != nil {
		labels["event_type"] = e.GetEventType().String()
	}

	if e.Deployment != nil {
		labels["deployment"] = e.GetDeployment()
	}

	if e.Job != nil {
		labels["job"] = e.GetJob()
	}

	if e.Index != nil {
		labels["index"] = e.GetIndex()
	}

	if e.Ip != nil {
		labels["ip"] = e.GetIp()
	}

	return labels
}