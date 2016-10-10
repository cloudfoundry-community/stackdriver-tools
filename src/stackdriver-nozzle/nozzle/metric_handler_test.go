package nozzle_test

import (
	"time"

	"errors"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricHandler", func() {
	var (
		subject       nozzle.MetricHandler
		metricAdapter *mocks.MetricAdapter
		labels        map[string]string
	)

	BeforeEach(func() {
		labels = map[string]string{"foo": "bar"}
		labelMaker := &mocks.LabelMaker{Labels: labels}
		metricAdapter = &mocks.MetricAdapter{}
		subject = nozzle.NewMetricHandler(labelMaker, metricAdapter)
	})

	It("creates the proper metrics for ContainerMetric", func() {
		eventTime := time.Now()

		diskBytesQuota := uint64(1073741824)
		instanceIndex := int32(0)
		cpuPercentage := 0.061651273460637
		diskBytes := uint64(164634624)
		memoryBytes := uint64(16601088)
		memoryBytesQuota := uint64(33554432)
		applicationId := "ee2aa52e-3c8a-4851-b505-0cb9fe24806e"
		timeStamp := eventTime.UnixNano()

		metricType := events.Envelope_ContainerMetric
		containerMetric := events.ContainerMetric{
			DiskBytesQuota:   &diskBytesQuota,
			InstanceIndex:    &instanceIndex,
			CpuPercentage:    &cpuPercentage,
			DiskBytes:        &diskBytes,
			MemoryBytes:      &memoryBytes,
			MemoryBytesQuota: &memoryBytesQuota,
			ApplicationId:    &applicationId,
		}

		envelope := &events.Envelope{
			EventType:       &metricType,
			ContainerMetric: &containerMetric,
			Timestamp:       &timeStamp,
		}

		err := subject.HandleEnvelope(envelope)
		Expect(err).To(BeNil())

		metrics := metricAdapter.PostedMetrics
		Expect(metrics).To(HaveLen(6))

		Expect(metrics).To(ContainElement(stackdriver.Metric{"diskBytesQuota", float64(1073741824), labels, eventTime}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{"instanceIndex", float64(0), labels, eventTime}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{"cpuPercentage", 0.061651273460637, labels, eventTime}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{"diskBytes", float64(164634624), labels, eventTime}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{"memoryBytes", float64(16601088), labels, eventTime}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{"memoryBytesQuota", float64(33554432), labels, eventTime}))
	})

	It("creates metric for CounterEvent", func() {
		eventTime := time.Now()

		eventType := events.Envelope_CounterEvent
		name := "counterName"
		total := uint64(123456)
		timeStamp := eventTime.UnixNano()

		event := events.CounterEvent{
			Name:  &name,
			Total: &total,
		}
		envelope := &events.Envelope{
			EventType:    &eventType,
			CounterEvent: &event,
			Timestamp:    &timeStamp,
		}

		err := subject.HandleEnvelope(envelope)
		Expect(err).To(BeNil())

		metrics := metricAdapter.PostedMetrics
		Expect(metrics).To(ConsistOf(stackdriver.Metric{
			"counterName",
			float64(123456),
			labels,
			eventTime,
		}))
	})

	It("returns the error from the metric adapter", func() {
		expectedErr := errors.New("fail")
		metricAdapter.PostMetricError = expectedErr

		eventType := events.Envelope_CounterEvent

		event := events.CounterEvent{}
		envelope := &events.Envelope{
			EventType:    &eventType,
			CounterEvent: &event,
		}

		err := subject.HandleEnvelope(envelope)
		Expect(err).To(Equal(expectedErr))
	})

	It("returns error when envelope contains unhandled event type", func() {
		eventType := events.Envelope_HttpStart
		envelope := &events.Envelope{
			EventType: &eventType,
		}

		err := subject.HandleEnvelope(envelope)

		Expect(err).NotTo(BeNil())
	})
})
