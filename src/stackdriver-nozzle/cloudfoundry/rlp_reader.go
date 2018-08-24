package cloudfoundry

import (
	"context"
	"crypto/tls"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/loggregator/plumbing/conversion"
	"github.com/cloudfoundry/sonde-go/events"
)

type ReverseLogProxyConfig struct {
	Address           string
	ShardID           string
	DeterministicName string
	TLSConfig         *tls.Config
}

type ReverseLogProxyHandler interface {
	HandleEvent(*events.Envelope) error
}

type ReverseLogProxy interface {
	Connect() (<-chan *events.Envelope, <-chan error)
}

type reverseLogProxy struct {
	config         *ReverseLogProxyConfig
	envelopeStream loggregator.EnvelopeStream
}

var allSelectors = []*loggregator_v2.Selector{
	{
		Message: &loggregator_v2.Selector_Log{
			Log: &loggregator_v2.LogSelector{},
		},
	},
	{
		Message: &loggregator_v2.Selector_Counter{
			Counter: &loggregator_v2.CounterSelector{},
		},
	},
	{
		Message: &loggregator_v2.Selector_Gauge{
			Gauge: &loggregator_v2.GaugeSelector{},
		},
	},
	{
		Message: &loggregator_v2.Selector_Timer{
			Timer: &loggregator_v2.TimerSelector{},
		},
	},
	{
		Message: &loggregator_v2.Selector_Event{
			Event: &loggregator_v2.EventSelector{},
		},
	},
}

func NewReverseLogProxy(config *ReverseLogProxyConfig) ReverseLogProxy {
	//TODO(evanbrown) Add WithEnvelopeStreamBuffer and alerter to track dropped envelopes
	streamConnector := loggregator.NewEnvelopeStreamConnector(
		config.Address,
		config.TLSConfig,
		//TODO(evanbrown): Do we want the stream logger?
		//loggregator.WithEnvelopeStreamLogger(loggr),
	)

	rx := streamConnector.Stream(context.Background(), &loggregator_v2.EgressBatchRequest{
		ShardId:           config.ShardID,
		DeterministicName: config.DeterministicName,
		Selectors:         allSelectors,
	})
	return reverseLogProxy{config, rx}
}

func (c reverseLogProxy) Connect() (<-chan *events.Envelope, <-chan error) {
	envelopes := make(chan *events.Envelope)
	errors := make(chan error)

	go func() {
		for {
			batch := c.envelopeStream()
			for _, e := range batch {
				for _, v1 := range conversion.ToV1(e) {
					envelopes <- v1
				}
			}
		}
	}()
	return envelopes, errors
}
