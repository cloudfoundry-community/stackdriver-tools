package filter_test

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	var (
		mockFirehoseHandler MockFirehoseHandler
	)

	BeforeEach(func() {
		mockFirehoseHandler = MockFirehoseHandler{}
	})

	It("can accept an empty filter and blocks all events", func() {
		emptyFilter := []string{}
		f, err := filter.New(mockFirehoseHandler, emptyFilter)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		mockFirehoseHandler.HandleEventFn = func(envelope *events.Envelope) error {
			Fail("Should not receive any events")
			return nil
		}

		SendAllEvents(f)
	})

	It("can accept a single event to filter", func() {
		singleFilter := []string{"Error"}
		f, err := filter.New(mockFirehoseHandler, singleFilter)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		mockFirehoseHandler.HandleEventFn = func(envelope *events.Envelope) error {
			Expect(envelope.GetEventType()).To(Equal(events.Envelope_Error))
			return nil
		}

		SendAllEvents(f)
	})

	It("rejects invalid events", func() {
		Fail("NYI")
	})
})

func SendAllEvents(filter firehose.FirehoseHandler) {
	for _, val := range events.Envelope_EventType_value {
		eventType := events.Envelope_EventType(val)
		event := events.Envelope{}
		event.EventType = &eventType

		filter.HandleEvent(&event)
	}
}

type MockFirehoseHandler struct {
	HandleEventFn func(envelope *events.Envelope) error
}

func (mfh MockFirehoseHandler) HandleEvent(envelope *events.Envelope) error {
	if mfh.HandleEventFn != nil {
		return mfh.HandleEventFn(envelope)
	} else {
		Fail("Unexpected call to HandleEvent")
	}
	return nil
}
