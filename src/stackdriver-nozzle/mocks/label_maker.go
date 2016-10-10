package mocks

import "github.com/cloudfoundry/sonde-go/events"

type LabelMaker struct{
	Labels map[string]string
}

func (lm *LabelMaker) Build(*events.Envelope) map[string]string {
	return lm.Labels
}
