package nozzle_test

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/serializer"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nozzle", func() {
	var (
		logAdapter    *mockLogAdapter
		metricAdapter *mockMetricAdapter
		subject       nozzle.Nozzle
	)

	BeforeEach(func() {
		logAdapter = newMockLogAdapter()
		metricAdapter = &mockMetricAdapter{}
		subject = nozzle.Nozzle{
			LogAdapter:    logAdapter,
			MetricAdapter: metricAdapter,
			Serializer:    serializer.NewSerializer(caching.NewCachingEmpty(), nil),
			Heartbeater:   &mockHeartbeater{},
		}
	})

	It("handles HttpStartStop", func() {
		eventType := events.Envelope_HttpStartStop
		envelope := &events.Envelope{EventType: &eventType}

		subject.HandleEvent(envelope)

		postedLog := logAdapter.postedLogs[0]
		Expect(postedLog.payload).To(Equal(envelope))
		Expect(postedLog.labels).To(Equal(map[string]string{
			"eventType": "HttpStartStop",
		}))
	})

	Context("metrics", func() {

		It("should post the value metric", func() {
			metricName := "memoryStats.lastGCPauseTimeNS"
			metricValue := float64(536182)
			metricType := events.Envelope_ValueMetric

			valueMetric := events.ValueMetric{
				Name:  &metricName,
				Value: &metricValue,
			}

			envelope := &events.Envelope{
				EventType:   &metricType,
				ValueMetric: &valueMetric,
			}

			err := subject.HandleEvent(envelope)
			Expect(err).To(BeNil())

			postedMetric := metricAdapter.postedMetrics[0]
			Expect(postedMetric.Name).To(Equal(metricName))
			Expect(postedMetric.Value).To(Equal(metricValue))
			Expect(postedMetric.Labels).To(Equal(map[string]string{
				"eventType": "ValueMetric",
			}))
		})

		It("should post the container metrics", func() {
			diskBytesQuota := uint64(1073741824)
			instanceIndex := int32(0)
			cpuPercentage := 0.061651273460637
			diskBytes := uint64(164634624)
			memoryBytes := uint64(16601088)
			memoryBytesQuota := uint64(33554432)
			applicationId := "ee2aa52e-3c8a-4851-b505-0cb9fe24806e"

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
			}

			err := subject.HandleEvent(envelope)
			Expect(err).To(BeNil())

			//TODO: add this test back when we finish the restructure
			//labels := map[string]string{
			//	"eventType":     "ContainerMetric",
			//	"applicationId": applicationId,
			//}
			Expect(len(metricAdapter.postedMetrics)).To(Equal(6))
			//Expect(metricAdapter.postedMetrics).To(ConsistOf(
			//	stackdriver.Metric{"diskBytesQuota", float64(1073741824), labels},
			//	stackdriver.Metric{"instanceIndex", float64(0), labels},
			//	stackdriver.Metric{"cpuPercentage", 0.061651273460637, labels},
			//	stackdriver.Metric{"diskBytes", float64(164634624), labels},
			//	stackdriver.Metric{"memoryBytes", float64(16601088), labels},
			//	stackdriver.Metric{"memoryBytesQuota", float64(33554432), labels},
			//))
		})

		It("returns error if client errors out", func() {
			expectedError := errors.New("fail")
			metricAdapter.postMetricError = expectedError
			metricType := events.Envelope_ContainerMetric
			envelope := &events.Envelope{
				EventType:   &metricType,
				ValueMetric: nil,
			}

			actualError := subject.HandleEvent(envelope)

			Expect(actualError).NotTo(BeNil())
			Expect(actualError).To(Equal(expectedError))
		})

		It("returns error if getting metric errors out", func() {
			const errMessage = "GetMetrics fail"
			mockSerializer := &mocks.MockSerializer{
				GetMetricsFn: func(*events.Envelope) ([]stackdriver.Metric, error) {
					return nil, errors.New(errMessage)
				},
				IsLogFn: func(*events.Envelope) bool {
					return false
				},
			}
			subject = nozzle.Nozzle{
				LogAdapter: nil,
				Serializer: mockSerializer,
			}

			envelope := &events.Envelope{}

			err := subject.HandleEvent(envelope)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal(errMessage))
		})
	})
})

type mockMetricAdapter struct {
	postedMetrics   []stackdriver.Metric
	postMetricError error
}

type mockLogAdapter struct {
	postedLogs []PostedLog

	mutex *sync.Mutex
}

func newMockLogAdapter() *mockLogAdapter {
	return &mockLogAdapter{
		postedLogs: []PostedLog{},
		mutex:      &sync.Mutex{},
	}
}

func (m *mockLogAdapter) PostLog(payload interface{}, labels map[string]string) {
	m.mutex.Lock()
	m.postedLogs = append(m.postedLogs, PostedLog{payload, labels})
	m.mutex.Unlock()
}

func (m *mockMetricAdapter) PostMetrics(metrics []stackdriver.Metric) error {
	m.postedMetrics = append(m.postedMetrics, metrics...)
	return m.postMetricError
}

type PostedLog struct {
	payload interface{}
	labels  map[string]string
}

type mockHeartbeater struct{}

func (mh *mockHeartbeater) Start()      {}
func (mh *mockHeartbeater) AddCounter() {}
func (mh *mockHeartbeater) Stop()       {}
