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

	It("ships something to the stackdriver client", func() {
		var postedEvent interface{}
		mockStackdriverClient.PostFn = func(e interface{}, _ map[string]string) {
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
		mockStackdriverClient.PostFn = func(e interface{}, _ map[string]string) {
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
		mockStackdriverClient.PostFn = func(e interface{}, sentLabels map[string]string) {
			labels = sentLabels
		}

		shippedEvent := map[string]interface{}{
			"event_type": "HttpStartStop",
			"foo":        "bar",
		}
		n := nozzle.Nozzle{StackdriverClient: mockStackdriverClient}

			n.ShipEvents(shippedEvent, "message")

		Expect(labels).To(Equal(map[string]string {
			"event_type": "HttpStartStop",
		}))
	})
})

type MockStackdriverClient struct {
	PostFn func(interface{}, map[string]string)
}

func (m *MockStackdriverClient) Post(payload interface{}, labels map[string]string) {
	if m.PostFn != nil {
		m.PostFn(payload, labels)
	}
}
