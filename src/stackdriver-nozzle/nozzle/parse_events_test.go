package nozzle_test

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseEvents", func() {
	It("handles empty events", func() {
		valid := []string{""}
		eventNames, err := nozzle.ParseEvents(valid)
		Expect(err).NotTo(HaveOccurred())
		Expect(eventNames).To(BeEmpty())
	})

	It("rejects invalid events", func() {
		invalidFilter := []string{"Error", "FakeEvent111"}
		eventNames, err := nozzle.ParseEvents(invalidFilter)
		Expect(err).To(HaveOccurred())
		Expect(eventNames).To(BeNil())
	})

	It("parses valid events", func() {
		valid := []string{"LogMessage", "ValueMetric"}
		eventNames, err := nozzle.ParseEvents(valid)
		Expect(err).NotTo(HaveOccurred())
		Expect(eventNames).To(ContainElement(events.Envelope_LogMessage))
		Expect(eventNames).To(ContainElement(events.Envelope_ValueMetric))
	})
})
