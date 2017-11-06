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

package nozzle_test

import (
	"errors"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nozzle", func() {
	var (
		subject     nozzle.Nozzle
		firehose    *mocks.FirehoseClient
		logSink     *mocks.Sink
		metricSink  *mocks.Sink
		heartbeater *mocks.Heartbeater
		logger      *mocks.MockLogger
	)

	BeforeEach(func() {
		firehose = mocks.NewFirehoseClient()
		logSink = &mocks.Sink{}
		metricSink = &mocks.Sink{}
		heartbeater = mocks.NewHeartbeater()
		logger = &mocks.MockLogger{}

		subject = nozzle.NewNozzle(logger, logSink, metricSink, heartbeater)
		subject.Start(firehose)
	})

	It("starts the heartbeater", func() {
		Expect(heartbeater.IsRunning()).To(Equal(true))
	})

	It("updates the heartbeater", func() {
		for _, value := range events.Envelope_EventType_value {
			eventType := events.Envelope_EventType(value)
			event := events.Envelope{EventType: &eventType}
			firehose.Messages <- &event
		}

		Eventually(func() int {
			return heartbeater.GetCount("nozzle.events")
		}).Should(Equal(len(events.Envelope_EventType_value)))
	})

	It("stops the heartbeater", func() {
		subject.Stop()
		Expect(heartbeater.IsRunning()).To(Equal(false))
	})

	It("does not receive errors", func() {
		for _, value := range events.Envelope_EventType_value {
			eventType := events.Envelope_EventType(value)
			event := events.Envelope{EventType: &eventType}
			firehose.Messages <- &event
		}
	})

	It("handles HttpStartStop event", func() {
		eventType := events.Envelope_HttpStartStop
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(logSink.LastEnvelope).Should(Equal(envelope))
	})

	It("handles LogMessage event", func() {
		eventType := events.Envelope_LogMessage
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(logSink.LastEnvelope).Should(Equal(envelope))
	})

	It("handles Error event", func() {
		eventType := events.Envelope_Error
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(logSink.LastEnvelope).Should(Equal(envelope))
	})

	It("handles ValueMetric event", func() {
		eventType := events.Envelope_ValueMetric
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(metricSink.LastEnvelope).Should(Equal(envelope))
	})

	It("handles ContainerMetric event", func() {
		eventType := events.Envelope_ContainerMetric
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(metricSink.LastEnvelope).Should(Equal(envelope))
	})

	It("handles CounterEvent event", func() {
		eventType := events.Envelope_CounterEvent
		envelope := &events.Envelope{EventType: &eventType}

		firehose.Messages <- envelope

		Eventually(metricSink.LastEnvelope).Should(Equal(envelope))
	})

	It("logs the firehose client errors", func() {
		err := errors.New("omg")
		go func() { firehose.Errs <- err }()

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.ERROR,
			Err:    err,
			Action: "firehose",
		}))
	})

	It("is resilient to multiple exists", func(done Done) {
		Expect(subject.Stop()).NotTo(HaveOccurred())
		Expect(subject.Stop()).To(HaveOccurred())
		Expect(subject.Stop()).To(HaveOccurred())

		Expect(heartbeater.IsRunning()).To(Equal(false))
		close(done)
	}, 0.2)
})
