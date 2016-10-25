package stackdriver_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsBuffer", func() {
	var (
		subject       stackdriver.MetricsBuffer
		errs          <-chan error
		metricAdapter *mocks.MetricAdapter
	)

	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}

		subject, errs = stackdriver.NewMetricsBuffer(5, metricAdapter)
	})

	It("acts as a passthrough with a buffer size of 1", func() {
		subject, errs = stackdriver.NewMetricsBuffer(1, metricAdapter)

		metric := &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(1))
		Expect(metricAdapter.PostedMetrics[0]).To(Equal(*metric))

		metric = &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(2))
		Expect(metricAdapter.PostedMetrics[1]).To(Equal(*metric))

		Consistently(errs).ShouldNot(Receive())
	})

	It("posts async", func() {
		postedNum := 0
		metricAdapter.PostMetricsFn = func(metrics []stackdriver.Metric) error {
			postedNum += 1
			time.Sleep(10 * time.Second)
			return nil
		}

		subject, errs = stackdriver.NewMetricsBuffer(1, metricAdapter)

		metric := &stackdriver.Metric{}
		go func() {
			subject.PostMetric(metric)
			subject.PostMetric(metric)
		}()

		Eventually(func() int {
			return postedNum
		}).Should(Equal(2))
	})

	It("only sends after the buffer size is reached", func() {
		subject.PostMetric(&stackdriver.Metric{Name: "a"})
		subject.PostMetric(&stackdriver.Metric{Name: "b"})
		subject.PostMetric(&stackdriver.Metric{Name: "c"})
		subject.PostMetric(&stackdriver.Metric{Name: "d"})

		Consistently(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(0))

		subject.PostMetric(&stackdriver.Metric{Name: "e"})
		Eventually(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(5))
		Consistently(errs).ShouldNot(Receive())
	})

	It("sends errors through the error channel", func() {
		subject, errs = stackdriver.NewMetricsBuffer(1, metricAdapter)

		expectedErr := errors.New("fail")
		metricAdapter.PostMetricError = expectedErr

		metric := &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(func() interface{} {
			return metricAdapter.PostedMetrics
		}).Should(HaveLen(1))

		var err error
		Eventually(errs).Should(Receive(&err))
		Expect(err).To(Equal(expectedErr))
	})

	It("posts current batch when encounters a duplicate", func() {
		subject.PostMetric(&stackdriver.Metric{Name: "a", Value: 1})
		subject.PostMetric(&stackdriver.Metric{Name: "b", Value: 2})
		subject.PostMetric(&stackdriver.Metric{Name: "a", Value: 2})
		Eventually(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(2))
		Expect(metricAdapter.PostedMetrics).To(Equal([]stackdriver.Metric{
			{Name: "a", Value: 1},
			{Name: "b", Value: 2},
		}))

		subject.PostMetric(&stackdriver.Metric{Name: "b", Value: 3})
		subject.PostMetric(&stackdriver.Metric{Name: "c"})
		subject.PostMetric(&stackdriver.Metric{Name: "d"})
		subject.PostMetric(&stackdriver.Metric{Name: "e"})
		Eventually(func() int {
			return len(metricAdapter.PostedMetrics)
		}).Should(Equal(7))
		Expect(metricAdapter.PostedMetrics).To(Equal([]stackdriver.Metric{
			{Name: "a", Value: 1},
			{Name: "b", Value: 2},
			{Name: "a", Value: 2},
			{Name: "b", Value: 3},
			{Name: "c"},
			{Name: "d"},
			{Name: "e"},
		}))
	})
})
