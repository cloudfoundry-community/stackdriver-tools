package metrics_pipeline_test

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	. "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/metrics_pipeline"
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

		router := NewRouter(metricAdapter, []events.Envelope_EventType{metricEvent}, logAdapter, []events.Envelope_EventType{logEvent})
		router.PostMetricEvents([]*messages.MetricEvent{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(metricAdapter.PostedMetricEvents).To(HaveLen(1))
		Expect(metricAdapter.PostMetricEventsCount).To(Equal(1))
		Expect(metricAdapter.PostedMetricEvents[0].Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		Expect(logAdapter.PostedLogs[0].Payload.(*messages.MetricEvent).Type).To(Equal(logEvent))
	})

	It("can route an event to two locations", func() {
		metricEvent := events.Envelope_ContainerMetric
		logEvent := events.Envelope_ValueMetric
		events := []events.Envelope_EventType{metricEvent, logEvent}

		router := NewRouter(metricAdapter, events, logAdapter, events)
		router.PostMetricEvents([]*messages.MetricEvent{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(metricAdapter.PostedMetricEvents).To(HaveLen(2))
		Expect(metricAdapter.PostMetricEventsCount).To(Equal(1))
		Expect(metricAdapter.PostedMetricEvents[0].Type).To(Equal(metricEvent))
		Expect(metricAdapter.PostedMetricEvents[1].Type).To(Equal(logEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(2))
		Expect(logAdapter.PostedLogs[0].Payload.(*messages.MetricEvent).Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs[1].Payload.(*messages.MetricEvent).Type).To(Equal(logEvent))
	})

	It("can translate Metric statements to Logs", func() {
		logEvent := events.Envelope_ValueMetric
		labels := map[string]string{"foo": "bar"}
		metric := &messages.Metric{
			Name:      "valueMetric",
			Value:     float64(123),
			EventTime: time.Now(),
			Unit:      "f",
		}
		metricEvent := &messages.MetricEvent{Type: logEvent, Labels: labels, Metrics: []*messages.Metric{metric}}
		router := NewRouter(nil, nil, logAdapter, []events.Envelope_EventType{logEvent})
		router.PostMetricEvents([]*messages.MetricEvent{metricEvent})
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		log := logAdapter.PostedLogs[0]
		Expect(log.Labels).To(Equal(labels))
		Expect(log.Payload).To(BeAssignableToTypeOf(&messages.MetricEvent{}))
		payload := log.Payload.(*messages.MetricEvent)
		Expect(payload.Metrics).To(Equal([]*messages.Metric{metric}))
		Expect(payload.Type).To(Equal(logEvent))
		Expect(payload.Labels).To(Equal(labels))
	})
})
