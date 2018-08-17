package cloudfoundry

type ReverseLogProxyHandler interface {
	HandleEvent(*events.Envelope) error
}

type ReverseLogProxy interface {
	Connect() (<-chan *events.Envelope, <-chan error)
}

type reverseLogProxy struct {
	cfConfig *cfclient.Config
	cfClient *cfclient.Client
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
	return &reverseLogProxy{cfConfig, cfClient}
}

func (c *reverseLogProxy) Connect() (<-chan *events.Envelope, <-chan error) {

}
