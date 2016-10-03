package filter_test

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/filter"

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

		for _, val := range events.Envelope_EventType_value {
			event := events.Envelope{}
			eventType := events.Envelope_EventType(val)
			event.EventType = &eventType
		}
	})
})

type MockFirehoseHandler struct {
	HandleEventFn func(envelope *events.Envelope) error
}

func (mfh MockFirehoseHandler) HandleEvent(envelope *events.Envelope) error {
	if mfh.HandleEventFn != nil {
		return mfh.HandleEventFn(envelope)
	}
	return nil
}
