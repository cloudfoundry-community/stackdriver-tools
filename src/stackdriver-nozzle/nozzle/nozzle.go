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

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry/sonde-go/events"
)

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

func (n *Nozzle) Start(firehose cloudfoundry.Firehose) (errs chan error, fhErrs <-chan error) {
	n.Heartbeater.Start()
	n.session = state{done: make(chan struct{}), running: true}

	errs = make(chan error)
	messages, fhErrs := firehose.Connect()
	go func() {
		for {
			select {
			case envelope := <-messages:
				n.Heartbeater.Increment("nozzle.events")
				err := n.handleEvent(envelope)
				if err != nil {
					errs <- err
				}
			case <-n.session.done:
				return
			}
		}
	}()

	return errs, fhErrs
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

func isMetric(envelope *events.Envelope) bool {
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric, events.Envelope_ContainerMetric, events.Envelope_CounterEvent:
		return true
	default:
		return false
	}
}
