package nozzle_test

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SinkFilter", func() {
	var (
		allEventTypes []events.Envelope_EventType
		sink          *mocks.Sink
	)

	BeforeEach(func() {
		allEventTypes = []events.Envelope_EventType{
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		}
		sink = &mocks.Sink{}
	})
	It("can accept an empty filter and blocks all events", func() {
		f, err := nozzle.NewFilterSink([]events.Envelope_EventType{}, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		for _, eventType := range allEventTypes {
			f.Receive(&events.Envelope{EventType: &eventType})
		}

		Expect(sink.HandledEnvelopes).To(BeEmpty())
	})

	It("can accept a single event", func() {
		f, err := nozzle.NewFilterSink([]events.Envelope_EventType{events.Envelope_LogMessage}, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		eventType := events.Envelope_LogMessage
		event := events.Envelope{EventType: &eventType}

		f.Receive(&event)
		Expect(sink.HandledEnvelopes).To(ContainElement(event))

	})

	It("can accept multiple events to filter", func() {
		f, err := nozzle.NewFilterSink([]events.Envelope_EventType{events.Envelope_ValueMetric, events.Envelope_LogMessage}, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		for _, eventType := range allEventTypes {
			f.Receive(&events.Envelope{EventType: &eventType})
		}

		Expect(sink.HandledEnvelopes).To(HaveLen(2))
	})

	It("requires a sink", func() {
		f, err := nozzle.NewFilterSink([]events.Envelope_EventType{}, nil)
		Expect(err).NotTo(BeNil())
		Expect(f).To(BeNil())
	})
})
