package nozzle_test

import (
	"errors"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nozzle", func() {

	var (
		mockStackdriverClient *MockStackdriverClient
		subject               nozzle.Nozzle
	)

	BeforeEach(func() {
		mockStackdriverClient = &MockStackdriverClient{}
		subject = nozzle.Nozzle{StackdriverClient: mockStackdriverClient}
	})

	Context("logging", func() {
		var envelope *events.Envelope

		BeforeEach(func() {
			eventType := events.Envelope_HttpStartStop
			envelope = &events.Envelope{
				EventType: &eventType,
			}
		})

		It("ships something to the stackdriver client", func() {
			var postedEvent interface{}
			mockStackdriverClient.PostLogFn = func(e interface{}, _ map[string]string) {
				postedEvent = e
			}

			subject.HandleEvent(envelope)

			Expect(postedEvent).To(Equal(nozzle.Envelope{envelope}))
		})

		It("ships multiple events", func() {
			count := 0
			mockStackdriverClient.PostLogFn = func(e interface{}, _ map[string]string) {
				count += 1
			}

			for i := 0; i < 10; i++ {
				subject.HandleEvent(envelope)
			}

			Expect(count).To(Equal(10))
		})

		It("creates labels from the event", func() {
			var labels map[string]string
			mockStackdriverClient.PostLogFn = func(e interface{}, sentLabels map[string]string) {
				labels = sentLabels
			}

			subject.HandleEvent(envelope)

			Expect(labels).To(Equal(map[string]string{
				"event_type": "HttpStartStop",
			}))
		})
	})

	Context("metrics", func() {
		var envelope *events.Envelope

		It("should post the metric", func() {
			var name string
			var value float64
			var labels map[string]string

			mockStackdriverClient.PostMetricFn = func(n string, v float64, l map[string]string) error {
				name = n
				value = v
				labels = l
				return nil
			}

			metricName := "memoryStats.lastGCPauseTimeNS"
			metricValue := float64(536182)
			metricType := events.Envelope_ValueMetric

			valueMetric := events.ValueMetric{
				Name:  &metricName,
				Value: &metricValue,
			}

			envelope = &events.Envelope{
				EventType:   &metricType,
				ValueMetric: &valueMetric,
			}

			err := subject.HandleEvent(envelope)
			Expect(err).To(BeNil())

			Expect(name).To(Equal(metricName))
			Expect(value).To(Equal(metricValue))
			Expect(labels).To(Equal(map[string]string{
				"event_type": "ValueMetric",
			}))
		})

		It("returns error if client errors out", func() {
			mockStackdriverClient.PostMetricFn = func(string, float64, map[string]string) error {
				return errors.New("fail")
			}

			err := subject.HandleEvent(envelope)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("fail"))
		})
	})
})

type MockStackdriverClient struct {
	PostLogFn    func(interface{}, map[string]string)
	PostMetricFn func(string, float64, map[string]string) error
}

func (m *MockStackdriverClient) PostLog(payload interface{}, labels map[string]string) {
	if m.PostLogFn != nil {
		m.PostLogFn(payload, labels)
	}
}

func (m *MockStackdriverClient) PostMetric(name string, value float64, labels map[string]string) error {
	if m.PostMetricFn != nil {
		return m.PostMetricFn(name, value, labels)
	}
	return nil
}
