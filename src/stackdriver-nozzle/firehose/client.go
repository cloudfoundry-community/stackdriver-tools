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
	StartListening(FirehoseHandler) error
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

func (c *client) StartListening(fh FirehoseHandler) error {
	cfConsumer := consumer.New(
		c.cfClient.Endpoint.DopplerEndpoint,
		&tls.Config{InsecureSkipVerify: c.cfConfig.SkipSslValidation},
		nil)

	refresher := CfClientTokenRefresh{cfClient: c.cfClient}
	cfConsumer.SetIdleTimeout(time.Duration(30) * time.Second)
	cfConsumer.RefreshTokenFrom(&refresher)
	messages, errs := cfConsumer.FirehoseWithoutReconnect(c.subscriptionID, "")

	for {
		select {
		case envelope := <-messages:
			err := fh.HandleEvent(envelope)
			if err != nil {
				c.logger.Error("handleEvent", err)
			}
		case err := <-errs:
			return err
		}
	}
}

type CfClientTokenRefresh struct {
	cfClient *cfclient.Client
}

func (ct *CfClientTokenRefresh) RefreshAuthToken() (string, error) {
	return ct.cfClient.GetToken(), nil
}
