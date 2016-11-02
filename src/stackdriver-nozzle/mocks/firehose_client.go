package mocks

import "github.com/cloudfoundry/sonde-go/events"

func NewFirehoseClient() *FirehoseClient {
	return &FirehoseClient{
		Messages: make(chan *events.Envelope),
		Errs:     make(chan error),
	}
}

type FirehoseClient struct {
	Messages chan *events.Envelope
	Errs     chan error
}

func (fc *FirehoseClient) Connect() (<-chan *events.Envelope, <-chan error) {
	return fc.Messages, fc.Errs
}

func (fc *FirehoseClient) SendEvents(eventTypes ...events.Envelope_EventType) {
	for i := range eventTypes {
		fc.Messages <- &events.Envelope{EventType: &eventTypes[i]}
	}
}
