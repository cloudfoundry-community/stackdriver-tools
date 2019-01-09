/*
 * Copyright 2019 Google Inc.
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
	"context"
	"crypto/tls"

	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/loggregator/plumbing/conversion"
	"github.com/cloudfoundry/sonde-go/events"
)

type ReverseLogProxyConfig struct {
	Address           string
	ShardID           string
	DeterministicName string
	TLSConfig         *tls.Config
}

// Wraps lager.Logger and adds methods to satisfy loggregator's logger type
type loggerWrapper struct {
	lager.Logger
}

func (l loggerWrapper) Panicf(s string, d ...interface{}) {
	data := lager.Data{}
	for i, j := range d {
		data[string(i)] = j
	}
	l.Fatal(s, nil, data)
}

func (l loggerWrapper) Printf(s string, d ...interface{}) {
	data := lager.Data{}
	for i, j := range d {
		data[string(i)] = j
	}
	l.Info(s, data)
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

func NewReverseLogProxy(config *ReverseLogProxyConfig, logger lager.Logger) ReverseLogProxy {
	wrapper := loggerWrapper{Logger: logger}
	streamConnector := loggregator.NewEnvelopeStreamConnector(
		config.Address,
		config.TLSConfig,
		loggregator.WithEnvelopeStreamLogger(wrapper),
	)

	rx := streamConnector.Stream(context.Background(), &loggregator_v2.EgressBatchRequest{
		ShardId:           config.ShardID,
		DeterministicName: config.DeterministicName,
		Selectors:         allSelectors,
	})
	return reverseLogProxy{config: config, envelopeStream: rx}
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
