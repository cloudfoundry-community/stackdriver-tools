package nozzle

import (
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseEvents", func() {
	It("handles empty events", func() {
		valid := []string{""}
		eventNames, err := ParseEvents(valid)
		Expect(err).NotTo(HaveOccurred())
		Expect(eventNames).To(BeEmpty())
	})

	It("rejects invalid events", func() {
		invalidFilter := []string{"Error", "FakeEvent111"}
		eventNames, err := ParseEvents(invalidFilter)
		Expect(err).To(HaveOccurred())
		Expect(eventNames).To(BeNil())
	})

	It("parses valid events", func() {
		valid := []string{"LogMessage", "ValueMetric"}
		eventNames, err := ParseEvents(valid)
		Expect(err).NotTo(HaveOccurred())
		Expect(eventNames).To(ContainElement(events.Envelope_LogMessage))
		Expect(eventNames).To(ContainElement(events.Envelope_ValueMetric))
	})
})
