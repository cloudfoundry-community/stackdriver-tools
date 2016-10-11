package nozzle_test

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogSink", func() {
	var (
		subject    nozzle.Sink
		labelMaker nozzle.LabelMaker
		logAdapter *mocks.LogAdapter
		labels     map[string]string
	)

	BeforeEach(func() {
		labels = map[string]string{"foo": "bar"}
		labelMaker = &mocks.LabelMaker{Labels: labels}
		logAdapter = &mocks.LogAdapter{}

		subject = nozzle.NewLogSink(labelMaker, logAdapter)
	})

	It("handles logs", func() {
		eventType := events.Envelope_HttpStartStop
		envelope := &events.Envelope{EventType: &eventType}

		subject.Receive(envelope)

		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		postedLog := logAdapter.PostedLogs[0]
		Expect(postedLog.Payload).To(Equal(envelope))
		Expect(postedLog.Labels).To(Equal(labels))
	})
})
