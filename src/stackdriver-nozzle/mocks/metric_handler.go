package mocks

import "github.com/cloudfoundry/sonde-go/events"

type MetricHandler struct {
	HandledEnvelopes []events.Envelope
	Error            error
}

func (mh *MetricHandler) HandleEnvelope(envelope *events.Envelope) error {
	mh.HandledEnvelopes = append(mh.HandledEnvelopes, *envelope)
	return mh.Error
}
