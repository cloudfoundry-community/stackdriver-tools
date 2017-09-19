package metrics_router_test

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/metrics_router"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var (
		metricAdapter *mocks.MetricAdapter
		logAdapter    *mocks.LogAdapter
	)
	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}
		logAdapter = &mocks.LogAdapter{}
	})
	It("can route events to a single location", func() {
		metricEvent := events.Envelope_ContainerMetric
		logEvent := events.Envelope_ValueMetric

		router := metrics_router.NewMetricsRouter(metricAdapter, []events.Envelope_EventType{metricEvent}, logAdapter, []events.Envelope_EventType{logEvent})
		err := router.PostMetrics([]messages.Metric{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(metricAdapter.PostedMetrics).To(HaveLen(1))
		Expect(metricAdapter.PostedMetrics[0].Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		Expect(logAdapter.PostedLogs[0].Payload.(messages.Metric).Type).To(Equal(logEvent))
	})

	It("can route an event to two locations", func() {
		metricEvent := events.Envelope_ContainerMetric
		logEvent := events.Envelope_ValueMetric
		events := []events.Envelope_EventType{metricEvent, logEvent}

		router := metrics_router.NewMetricsRouter(metricAdapter, events, logAdapter, events)
		err := router.PostMetrics([]messages.Metric{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(metricAdapter.PostedMetrics).To(HaveLen(2))
		Expect(metricAdapter.PostedMetrics[0].Type).To(Equal(metricEvent))
		Expect(metricAdapter.PostedMetrics[1].Type).To(Equal(logEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(2))
		Expect(logAdapter.PostedLogs[0].Payload.(messages.Metric).Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs[1].Payload.(messages.Metric).Type).To(Equal(logEvent))
	})

	It("can translate Metric statements to Logs", func() {
		logEvent := events.Envelope_ValueMetric
		metric := messages.Metric{
			Labels:    map[string]string{"foo": "bar"},
			Type:      logEvent,
			Name:      "valueMetric",
			Value:     float64(123),
			EventTime: time.Now(),
			Unit:      "f",
		}
		router := metrics_router.NewMetricsRouter(nil, nil, logAdapter, []events.Envelope_EventType{logEvent})
		router.PostMetrics([]messages.Metric{metric})
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		log := logAdapter.PostedLogs[0]
		Expect(log.Labels).To(Equal(metric.Labels))
		Expect(log.Payload).To(BeAssignableToTypeOf(messages.Metric{}))
		payload := log.Payload.(messages.Metric)
		Expect(payload).To(Equal(metric))
	})
})
