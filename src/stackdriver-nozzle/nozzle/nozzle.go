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

package nozzle

import (
	"errors"
	"sync"

	"expvar"

	"code.cloudfoundry.org/diodes"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gorilla/websocket"
)

const bufferSize = 30000 // 1k messages/second * 30 seconds

type Nozzle interface {
	Start(firehose cloudfoundry.Firehose)
	Stop() error
}

var (
	firehoseErrs *expvar.Map

	firehoseErrEmpty                *expvar.Int
	firehoseErrUnknown              *expvar.Int
	firehoseErrCloseNormal          *expvar.Int
	firehoseErrClosePolicyViolation *expvar.Int
	firehoseErrCloseUnknown         *expvar.Int

	nozzleEventsTotal    *expvar.Int
	nozzleEventsDropped  *expvar.Int
	nozzleEventsReceived *expvar.Int
)

func init() {
	firehoseErrs = expvar.NewMap("nozzle.firehose.errors")

	firehoseErrEmpty = &expvar.Int{}
	firehoseErrUnknown = &expvar.Int{}
	firehoseErrCloseNormal = &expvar.Int{}
	firehoseErrClosePolicyViolation = &expvar.Int{}
	firehoseErrCloseUnknown = &expvar.Int{}

	firehoseErrs.Set("empty", firehoseErrEmpty)
	firehoseErrs.Set("unknown", firehoseErrUnknown)
	firehoseErrs.Set("close_normal_closure", firehoseErrCloseNormal)
	firehoseErrs.Set("close_policy_violation", firehoseErrClosePolicyViolation)
	firehoseErrs.Set("close_unknown", firehoseErrCloseUnknown)

	nozzleEventsTotal = expvar.NewInt("nozzle.events.total")
	nozzleEventsDropped = expvar.NewInt("nozzle.events.dropped")
	nozzleEventsReceived = expvar.NewInt("nozzle.events.received")
}

type nozzle struct {
	logSink    Sink
	metricSink Sink

	logger lager.Logger

	session state
}

type state struct {
	sync.Mutex
	done    chan struct{}
	running bool
}

func NewNozzle(logger lager.Logger, logSink Sink, metricSink Sink) Nozzle {
	return &nozzle{
		logSink:    logSink,
		metricSink: metricSink,
		logger:     logger,
	}
}

func (n *nozzle) Start(firehose cloudfoundry.Firehose) {
	n.session = state{done: make(chan struct{}), running: true}

	messages, fhErrInternal := firehose.Connect()

	// Drain and report errors from firehose
	go func() {
		for err := range fhErrInternal {
			if err == nil {
				// Ignore empty errors. Customers observe a flooding of empty errors from firehose.
				firehoseErrEmpty.Add(1)
				continue
			}

			go n.handleFirehoseError(err)
		}
	}()

	buffer := diodes.NewPoller(diodes.NewOneToOne(bufferSize, diodes.AlertFunc(func(missed int) {
		nozzleEventsDropped.Add(int64(missed))
		nozzleEventsTotal.Add(int64(missed))
	})))

	// Drain messages from the firehose and place them into the ring buffer
	go func() {
		for {
			select {
			case envelope := <-messages:
				buffer.Set(diodes.GenericDataType(envelope))
			case <-n.session.done:
				return
			}
		}
	}()

	// Drain the ring buffer of firehose events to send through the nozzle
	go func() {
		for {
			unsafeEvent := buffer.Next()
			if unsafeEvent != nil {
				var event = (*events.Envelope)(unsafeEvent)
				n.handleEvent(event)
			}
		}
	}()
}

func (n *nozzle) Stop() error {
	n.session.Lock()
	defer n.session.Unlock()

	if !n.session.running {
		return errors.New("nozzle is not running")
	}
	close(n.session.done)
	n.session.running = false

	return nil
}

func (n *nozzle) handleEvent(envelope *events.Envelope) {
	nozzleEventsReceived.Add(1)
	nozzleEventsTotal.Add(1)
	if isMetric(envelope) {
		n.metricSink.Receive(envelope)
	} else {
		n.logSink.Receive(envelope)
	}
}

func (n *nozzle) handleFirehoseError(err error) {
	if err == consumer.ErrMaxRetriesReached {
		n.logger.Fatal("firehose", err)
	} else {
		n.logger.Error("firehose", err)
	}

	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		firehoseErrUnknown.Add(1)
		return
	}

	switch closeErr.Code {
	case websocket.CloseNormalClosure:
		firehoseErrCloseNormal.Add(1)
	case websocket.ClosePolicyViolation:
		firehoseErrClosePolicyViolation.Add(1)
	default:
		firehoseErrCloseUnknown.Add(1)
	}
}

func isMetric(envelope *events.Envelope) bool {
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric, events.Envelope_ContainerMetric, events.Envelope_CounterEvent:
		return true
	default:
		return false
	}
}
