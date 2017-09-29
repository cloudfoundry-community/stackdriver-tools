/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cloudfoundry

import (
	"crypto/tls"
	"fmt"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
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
	return cfConsumer.Firehose(c.subscriptionID, "")
}

type cfClientTokenRefresh struct {
	cfClient *cfclient.Client
}

func (ct *cfClientTokenRefresh) RefreshAuthToken() (token string, err error) {
	// GetToken() doesn't return an error if there is a problem retrieving the
	// refresh token. If the token is an empty string, that is an error and
	// we return it. The downstream client should call firehose.Connect to get
	// a new connection.
	//
	// TODO: Track https://github.com/cloudfoundry-community/go-cfclient/issues/34 for
	// updates on proper refresh token handling.
	token, err = ct.cfClient.GetToken()
	if token == "" && err == nil {
		err = fmt.Errorf("Fatal: error getting refresh token")
	}
	return
}
