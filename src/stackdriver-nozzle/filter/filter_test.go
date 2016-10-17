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
		fhClient *mocks.FirehoseClient
	)

	BeforeEach(func() {
		fhClient = mocks.NewFirehoseClient()
	})

	It("can accept an empty filter and blocks all events", func() {
		emptyFilter := []string{}
		f, err := filter.New(fhClient, emptyFilter)
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
		f, err := filter.New(fhClient, singleFilter)
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
		f, err := filter.New(fhClient, multiFilter)
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
		f, err := filter.New(fhClient, invalidFilter)
		Expect(err).NotTo(BeNil())
		Expect(f).To(BeNil())
	})
})
