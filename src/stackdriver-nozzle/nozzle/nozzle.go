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
	"strings"
	"sync"

	"code.cloudfoundry.org/diodes"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gorilla/websocket"
)

const bufferSize = 30000 // 1k messages/second * 30 seconds

type PostMetricError struct {
	Errors []error
}

func (e *PostMetricError) Error() string {
	errors := []string{}
	for _, err := range e.Errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, "\n")
}

type Nozzle struct {
	LogSink    Sink
	MetricSink Sink

	Heartbeater heartbeat.Heartbeater

	session state
}

type state struct {
	sync.Mutex
	done    chan struct{}
	running bool
}

func (n *Nozzle) Start(firehose cloudfoundry.Firehose) (errs chan error, firehoseErrs chan error) {
	n.Heartbeater.Start()
	n.session = state{done: make(chan struct{}), running: true}

	errs = make(chan error)
	firehoseErrs = make(chan error)
	messages, fhErrInternal := firehose.Connect()

	buffer := diodes.NewPoller(diodes.NewOneToOne(bufferSize, diodes.AlertFunc(func(missed int) {
		// TODO(jrjohnson): Introduce Heartbeater.Increment(event, count)
		for i := 0; i < missed; i++ {
			n.Heartbeater.Increment("nozzle.events.dropped")
		}
	})))

	// Drain from the firehose
	go func() {
		for {
			select {
			case envelope := <-messages:
				buffer.Set(diodes.GenericDataType(envelope))
			case <-n.session.done:
				return
			case fhErr := <-fhErrInternal:
				n.handleFirehoseError(fhErr)
				firehoseErrs <- fhErr
			}
		}
	}()

	// Send firehose events through the nozzle
	go func() {
		for {
			unsafeEvent := buffer.Next()
			if unsafeEvent != nil {
				var event = (*events.Envelope)(unsafeEvent)

				if err := n.handleEvent(event); err != nil {
					errs <- err
				}
			}
		}
	}()

	return errs, firehoseErrs
}

func (n *Nozzle) Stop() error {
	n.session.Lock()
	defer n.session.Unlock()

	if !n.session.running {
		return errors.New("nozzle is not running")
	}
	n.Heartbeater.Stop()
	close(n.session.done)
	n.session.running = false

	return nil
}

func (n *Nozzle) handleEvent(envelope *events.Envelope) error {
	n.Heartbeater.Increment("nozzle.events")
	if isMetric(envelope) {
		if err := n.MetricSink.Receive(envelope); err != nil {
			return err
		}
	} else {
		if err := n.LogSink.Receive(envelope); err != nil {
			return err
		}
	}

	return nil
}

func (n *Nozzle) handleFirehoseError(err error) {
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		n.Heartbeater.Increment("firehose.errors.unknwon")
		return
	}

	switch closeErr.Code {
	case websocket.CloseNormalClosure:
		n.Heartbeater.Increment("firehose.errors.close.normal_close")
	case websocket.ClosePolicyViolation:
		n.Heartbeater.Increment("firehose.errors.close.policy_violation")
	default:
		n.Heartbeater.Increment("firehose.errors.close.unknown")
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
