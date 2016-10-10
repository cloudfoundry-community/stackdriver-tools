package serializer_test

import (
	"time"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/serializer"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Serializer", func() {
	var (
		subject serializer.Serializer
		logger  *mocks.MockLogger
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		subject = serializer.NewSerializer(caching.NewCachingEmpty(), logger)
	})

	Context("GetMetrics", func() {
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

			metrics, err := subject.GetMetrics(envelope)
			Expect(err).To(BeNil())

			Expect(metrics).To(HaveLen(6))

			labels := map[string]string{
				"eventType":     "ContainerMetric",
				"applicationId": applicationId,
			}

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

			metrics, err := subject.GetMetrics(envelope)
			Expect(err).To(BeNil())

			labels := map[string]string{
				"eventType": "CounterEvent",
			}
			Expect(metrics).To(ConsistOf(stackdriver.Metric{
				"counterName",
				float64(123456),
				labels,
				eventTime,
			}))
		})

		It("returns error when envelope contains unhandled event type", func() {
			eventType := events.Envelope_HttpStart
			envelope := &events.Envelope{
				EventType: &eventType,
			}
			_, err := subject.GetMetrics(envelope)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("isLog", func() {
		It("HttpStartStop is log", func() {
			eventType := events.Envelope_HttpStartStop

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())
		})

		It("LogMessage is log", func() {
			eventType := events.Envelope_LogMessage

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())
		})

		It("ValueMetric is *NOT* log", func() {
			eventType := events.Envelope_ValueMetric

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())
		})

		It("CounterEvent is *NOT* log", func() {
			eventType := events.Envelope_CounterEvent

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())

		})

		It("Error is log", func() {
			eventType := events.Envelope_Error

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())

		})

		It("ContainerMetric is *NOT* log", func() {
			eventType := events.Envelope_ContainerMetric

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())

		})
	})
})
