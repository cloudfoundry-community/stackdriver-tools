package nozzle_test

import (
	"errors"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
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
		errs        chan error
		fhErrs      <-chan error
	)

	BeforeEach(func() {
		firehose = mocks.NewFirehoseClient()
		logSink = &mocks.Sink{}
		metricSink = &mocks.Sink{}
		heartbeater = mocks.New()

		subject = nozzle.Nozzle{
			LogSink:     logSink,
			MetricSink:  metricSink,
			Heartbeater: heartbeater,
		}
		errs, fhErrs = subject.Start(firehose)
	})

	It("starts the heartbeater", func() {
		Expect(heartbeater.Started).To(Equal(true))
	})

	It("updates the heartbeater", func() {
		for _, value := range events.Envelope_EventType_value {
			eventType := events.Envelope_EventType(value)
			event := events.Envelope{EventType: &eventType}
			firehose.Messages <- &event
		}

		Eventually(func() int {
			return heartbeater.Counters["nozzle.events"]
		}).Should(Equal(len(events.Envelope_EventType_value)))
	})

	It("stops the heartbeater", func() {
		subject.Stop()
		Expect(heartbeater.Started).To(Equal(false))
	})

	It("does not receive errors", func() {
		for _, value := range events.Envelope_EventType_value {
			eventType := events.Envelope_EventType(value)
			event := events.Envelope{EventType: &eventType}
			firehose.Messages <- &event
		}

		Consistently(errs).ShouldNot(Receive())
		Consistently(fhErrs).ShouldNot(Receive())
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

	It("returns error if handler errors out", func() {
		expectedError := errors.New("fail")
		metricSink.Error = expectedError
		metricType := events.Envelope_ValueMetric
		envelope := &events.Envelope{
			EventType:   &metricType,
			ValueMetric: nil,
		}

		firehose.Messages <- envelope
		Eventually(errs).Should(Receive(Equal(expectedError)))
	})

	It("returns the firehose client errors", func() {
		err := errors.New("omg")
		go func() { firehose.Errs <- err }()

		Eventually(fhErrs).Should(Receive(Equal(err)))
	})
})
