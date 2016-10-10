package nozzle_test

import (
	"errors"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/serializer"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nozzle", func() {
	var (
		logHandler    *mocks.Handler
		metricHandler *mocks.Handler
		subject       nozzle.Nozzle
	)

	BeforeEach(func() {
		logHandler = &mocks.Handler{}
		metricHandler = &mocks.Handler{}
		subject = nozzle.Nozzle{
			LogHandler:    logHandler,
			MetricHandler: metricHandler,
			Serializer:    serializer.NewSerializer(caching.NewCachingEmpty(), nil),
			Heartbeater:   &mockHeartbeater{},
		}
	})

	It("handles HttpStartStop event", func() {
		eventType := events.Envelope_HttpStartStop
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := logHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("handles LogMessage event", func() {
		eventType := events.Envelope_LogMessage
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := logHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("handles Error event", func() {
		eventType := events.Envelope_Error
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := logHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("handles ValueMetric event", func() {
		eventType := events.Envelope_ValueMetric
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := metricHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("handles ContainerMetric event", func() {
		eventType := events.Envelope_ContainerMetric
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := metricHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("handles CounterEvent event", func() {
		eventType := events.Envelope_CounterEvent
		envelope := &events.Envelope{EventType: &eventType}

		err := subject.HandleEvent(envelope)
		Expect(err).To(BeNil())

		handledEnvelope := metricHandler.HandledEnvelopes[0]
		Expect(handledEnvelope).To(Equal(*envelope))
	})

	It("returns error if handler errors out", func() {
		expectedError := errors.New("fail")
		metricHandler.Error = expectedError
		metricType := events.Envelope_ValueMetric
		envelope := &events.Envelope{
			EventType:   &metricType,
			ValueMetric: nil,
		}

		actualError := subject.HandleEvent(envelope)

		Expect(actualError).NotTo(BeNil())
		Expect(actualError).To(Equal(expectedError))
	})
})

type mockHeartbeater struct{}

func (mh *mockHeartbeater) Start()      {}
func (mh *mockHeartbeater) AddCounter() {}
func (mh *mockHeartbeater) Stop()       {}
