package filter_test

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	var (
		fhClient    *mocks.FirehoseClient
		heartbeater *mocks.Heartbeater
	)

	BeforeEach(func() {
		fhClient = mocks.NewFirehoseClient()
		heartbeater = mocks.New()
	})

	It("can accept an empty filter and blocks all events", func() {
		emptyFilter := []string{}
		f, err := filter.New(fhClient, emptyFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		go fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_Error,
			events.Envelope_ContainerMetric,
		)

		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("can accept a single event to filter", func() {
		singleFilter := []string{"Error"}
		f, err := filter.New(fhClient, singleFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		eventType := events.Envelope_Error
		event := &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		go fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)
		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("can accept multiple events to filter", func() {
		multiFilter := []string{"Error", "LogMessage"}
		f, err := filter.New(fhClient, multiFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		eventType := events.Envelope_Error
		event := &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		eventType = events.Envelope_LogMessage
		event = &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		go fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)
		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("rejects invalid events", func() {
		invalidFilter := []string{"Error", "FakeEvent111"}
		f, err := filter.New(fhClient, invalidFilter, heartbeater)
		Expect(err).NotTo(BeNil())
		Expect(f).To(BeNil())
	})

	It("increments the heartbeater", func() {
		multiFilter := []string{"Error", "LogMessage"}
		f, _ := filter.New(fhClient, multiFilter, heartbeater)
		messages, _ := f.Connect()

		go func() {
			for range messages {
			}
		}()

		fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
		)

		Expect(heartbeater.Counters["filter.events"]).To(Equal(3))

		fhClient.SendEvents(
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)

		Expect(heartbeater.Counters["filter.events"]).To(Equal(7))
	})
})
