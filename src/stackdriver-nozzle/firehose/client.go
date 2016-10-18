package firehose

import (
	"crypto/tls"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
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
	subscriptionID string
}

func NewClient(cfConfig *cfclient.Config, cfClient *cfclient.Client, subscriptionID string) Client {
	return &client{cfConfig, cfClient, subscriptionID}
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
