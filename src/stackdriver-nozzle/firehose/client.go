package firehose

import (
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/firehose-to-syslog/eventRouting"
	"github.com/cloudfoundry-community/firehose-to-syslog/firehoseclient"
	"github.com/cloudfoundry-community/firehose-to-syslog/logging"
	"github.com/cloudfoundry-community/go-cfclient"
)

type Client interface {
	StartListening(nozzle logging.Logging) error
}

type client struct {
	cfConfig *cfclient.Config
}

func NewClient(apiAddress, username, password string, skipSSLValidation bool) Client {
	return &client{
		&cfclient.Config{
			ApiAddress:        apiAddress,
			Username:          username,
			Password:          password,
			SkipSslValidation: skipSSLValidation}}
}

func (fc *client) StartListening(nozzle logging.Logging) error {
	cfClient := cfclient.NewClient(fc.cfConfig)

	cachingClient := caching.NewCachingEmpty()

	eventRouter := eventRouting.NewEventRouting(cachingClient, nozzle)

	//err := eventRouter.SetupEventRouting("LogMessage,ValueMetric,HttpStartStop,CounterEvent,Error,ContainerMetric")
	err := eventRouter.SetupEventRouting("HttpStartStop")
	if err != nil {
		return err
	}

	if nozzle.Connect() {
		firehoseConfig := &firehoseclient.FirehoseConfig{
			TrafficControllerURL:   cfClient.Endpoint.DopplerEndpoint,
			InsecureSSLSkipVerify:  fc.cfConfig.SkipSslValidation,
			IdleTimeoutSeconds:     30,
			FirehoseSubscriptionID: "stackdriver-nozzle",
		}

		firehoseClient := firehoseclient.NewFirehoseNozzle(cfClient, eventRouter, firehoseConfig)

		firehoseClient.Start()
	}
	return nil
}
