package firehose

import (
	"crypto/tls"
	"time"

	"errors"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type FirehoseHandler interface {
	HandleEvent(*events.Envelope) error
}

type Client interface {
	Connect() (<-chan *events.Envelope, <-chan error)
}

type client struct {
	cfConfig       *cfclient.Config
	cfClient       *cfclient.Client
	logger         lager.Logger
	subscriptionID string
}

func NewClient(cfConfig *cfclient.Config, cfClient *cfclient.Client, logger lager.Logger, subscriptionID string) Client {
	if cfConfig == nil || cfClient == nil {
		logger.Fatal("firehoseClient", errors.New("cfClient and cfConfig required"))
	}
	return &client{cfConfig, cfClient, logger, subscriptionID}
}

func (c *client) Connect() (<-chan *events.Envelope, <-chan error) {
	cfConsumer := consumer.New(
		c.cfClient.Endpoint.DopplerEndpoint,
		&tls.Config{InsecureSkipVerify: c.cfConfig.SkipSslValidation},
		nil)

	refresher := cfClientTokenRefresh{cfClient: c.cfClient}
	cfConsumer.SetIdleTimeout(time.Duration(30) * time.Second)
	cfConsumer.RefreshTokenFrom(&refresher)
	return cfConsumer.FirehoseWithoutReconnect(c.subscriptionID, "")
}

type cfClientTokenRefresh struct {
	cfClient *cfclient.Client
}

func (ct *cfClientTokenRefresh) RefreshAuthToken() (string, error) {
	return ct.cfClient.GetToken(), nil
}
