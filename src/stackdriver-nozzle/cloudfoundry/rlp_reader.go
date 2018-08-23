package cloudfoundry

import (
	"context"
	"log"
	"os"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/loggregator/plumbing/conversion"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
)

type ReverseLogProxyHandler interface {
	HandleEvent(*events.Envelope) error
}

type ReverseLogProxy interface {
	Connect() (<-chan *events.Envelope, <-chan error)
}

type reverseLogProxy struct {
	cfConfig       *cfclient.Config
	cfClient       *cfclient.Client
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

func NewReverseLogProxy(cfConfig *cfclient.Config, cfClient *cfclient.Client) ReverseLogProxy {
	tlsConfig, err := loggregator.NewEgressTLSConfig(
		//TODO(evanbrown): get from app config
		os.Getenv("CA_CERT_PATH"),
		os.Getenv("CERT_PATH"),
		os.Getenv("KEY_PATH"),
	)
	if err != nil {
		log.Fatal("Could not create TLS config", err)
	}

	streamConnector := loggregator.NewEnvelopeStreamConnector(
		//TODO(evanbrown): get from app config
		os.Getenv("LOGS_API_ADDR"),
		tlsConfig,
		//TODO(evanbrown): Do we want the stream logger?
		//loggregator.WithEnvelopeStreamLogger(loggr),
	)

	rx := streamConnector.Stream(context.Background(), &loggregator_v2.EgressBatchRequest{
		//TODO(evanbrown): Set ShardID and DN
		//ShardId:           "test",
		//DeterministicName: os.Getenv("DET_NAME"),
		Selectors: allSelectors,
	})
	return reverseLogProxy{cfConfig, cfClient, rx}
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
