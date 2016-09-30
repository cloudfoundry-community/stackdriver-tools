package nozzle_test

import (
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nozzle", func() {

	var (
		mockStackdriverClient *MockStackdriverClient
	)

	BeforeEach(func() {
		mockStackdriverClient = &MockStackdriverClient{}
	})

	Context("logging", func() {
		It("ships something to the stackdriver client", func() {
			var postedEvent interface{}
			mockStackdriverClient.PostLogFn = func(e interface{}, _ map[string]string) {
				postedEvent = e
			}

			shippedEvent := map[string]interface{}{
				"event_type": "HttpStartStop",
				"foo":        "bar",
			}

			n := nozzle.Nozzle{StackdriverClient: mockStackdriverClient}
			n.ShipEvents(shippedEvent, "message")

			Expect(postedEvent).To(Equal(shippedEvent))
		})

		It("ships multiple events", func() {
			count := 0
			mockStackdriverClient.PostLogFn = func(e interface{}, _ map[string]string) {
				count += 1
			}

			shippedEvent := map[string]interface{}{
				"event_type": "HttpStartStop",
				"foo":        "bar",
			}
			n := nozzle.Nozzle{StackdriverClient: mockStackdriverClient}

			for i := 0; i < 10; i++ {
				n.ShipEvents(shippedEvent, "message")
			}

			Expect(count).To(Equal(10))
		})

		It("creates labels from the event", func() {
			var labels map[string]string
			mockStackdriverClient.PostLogFn = func(e interface{}, sentLabels map[string]string) {
				labels = sentLabels
			}

			shippedEvent := map[string]interface{}{
				"event_type": "HttpStartStop",
				"foo":        "bar",
			}
			n := nozzle.Nozzle{StackdriverClient: mockStackdriverClient}

			n.ShipEvents(shippedEvent, "message")

			Expect(labels).To(Equal(map[string]string{
				"event_type": "HttpStartStop",
			}))
		})
	})

	Context("metrics", func() {
		It("should post the metric", func() {
			var name string
			var value float64
			mockStackdriverClient.PostMetricFn = func(n string, v float64) error {
				name = n
				value = v
				return nil
			}

			shippedEvent := map[string]interface{}{
				"event_type": "ValueMetric",
				"name":       "memoryStats.lastGCPauseTimeNS",
				"value":      536182.,
			}

			n := nozzle.Nozzle{StackdriverClient: mockStackdriverClient}
			n.ShipEvents(shippedEvent, "message")

			Expect(name).To(Equal("memoryStats.lastGCPauseTimeNS"))
			Expect(value).To(Equal(536182.))
		})
	})
})

type MockStackdriverClient struct {
	PostLogFn    func(interface{}, map[string]string)
	PostMetricFn func(string, float64) error
}

func (m *MockStackdriverClient) PostLog(payload interface{}, labels map[string]string) {
	if m.PostLogFn != nil {
		m.PostLogFn(payload, labels)
	}
}

func (m *MockStackdriverClient) PostMetric(name string, value float64) error {
	if m.PostMetricFn != nil {
		return m.PostMetricFn(name, value)
	}
	return nil
}
