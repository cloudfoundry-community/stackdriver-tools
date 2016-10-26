package cloudfoundry

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

type Firehose interface {
	Connect() (<-chan *events.Envelope, <-chan error)
}

type firehose struct {
	cfConfig       *cfclient.Config
	cfClient       *cfclient.Client
	subscriptionID string
}

func NewFirehose(cfConfig *cfclient.Config, cfClient *cfclient.Client, subscriptionID string) Firehose {
	return &firehose{cfConfig, cfClient, subscriptionID}
}

func (c *firehose) Connect() (<-chan *events.Envelope, <-chan error) {
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
