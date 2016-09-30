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
	StartListening(FirehoseHandler) error
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

func (c *client) StartListening(fh FirehoseHandler) error {
	cfConfig := &cfclient.Config{
		ApiAddress:        c.cfConfig.ApiAddress,
		Username:          c.cfConfig.Username,
		Password:          c.cfConfig.Password,
		SkipSslValidation: c.cfConfig.SkipSslValidation}
	cfClient := cfclient.NewClient(cfConfig)

	cfConsumer := consumer.New(
		cfClient.Endpoint.DopplerEndpoint,
		&tls.Config{InsecureSkipVerify: c.cfConfig.SkipSslValidation},
		nil)

	refresher := CfClientTokenRefresh{cfClient: cfClient}
	cfConsumer.SetIdleTimeout(time.Duration(30) * time.Second)
	cfConsumer.RefreshTokenFrom(&refresher)
	messages, errs := cfConsumer.FirehoseWithoutReconnect("test", "")

	for {
		select {
		case envelope := <-messages:
			err := fh.HandleEvent(envelope)
			if err != nil {
				return err
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
