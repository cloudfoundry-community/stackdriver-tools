package stackdriver_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
)

var _ = Describe("MetricsBuffer", func() {
	var (
		metricAdapter *mocks.MetricAdapter
	)

	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}

		//subject, errs = stackdriver.NewMetricsBuffer(5, metricAdapter)
	})

	It("acts as a passthrough with a buffer size of 1", func() {
		subject, errs := stackdriver.NewMetricsBuffer(1, metricAdapter)

		metric := &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(1))
		Expect(metricAdapter.GetPostedMetrics()[0]).To(Equal(*metric))

		metric = &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(2))
		Expect(metricAdapter.GetPostedMetrics()[1]).To(Equal(*metric))

		Consistently(errs).ShouldNot(Receive())
	})

	It("posts async", func() {
		var mutex sync.Mutex
		postedNum := 0
		metricAdapter.PostMetricsFn = func(metrics []stackdriver.Metric) error {
			mutex.Lock()
			postedNum += 1
			mutex.Unlock()

			time.Sleep(10 * time.Second)
			return nil
		}

		subject, _ := stackdriver.NewMetricsBuffer(1, metricAdapter)

		metric := &stackdriver.Metric{}
		go func() {
			subject.PostMetric(metric)
			subject.PostMetric(metric)
		}()

		Eventually(func() int {
			mutex.Lock()
			defer mutex.Unlock()

			return postedNum
		}).Should(Equal(2))
	})

	It("only sends after the buffer size is reached", func() {
		subject, errs := stackdriver.NewMetricsBuffer(5, metricAdapter)

		subject.PostMetric(&stackdriver.Metric{Name: "a"})
		subject.PostMetric(&stackdriver.Metric{Name: "b"})
		subject.PostMetric(&stackdriver.Metric{Name: "c"})
		subject.PostMetric(&stackdriver.Metric{Name: "d"})
		Consistently(metricAdapter.GetPostedMetrics).Should(HaveLen(0))

		subject.PostMetric(&stackdriver.Metric{Name: "e"})
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(5))

		Consistently(errs).ShouldNot(Receive())
	})

	It("sends errors through the error channel", func() {
		subject, errs := stackdriver.NewMetricsBuffer(1, metricAdapter)

		expectedErr := errors.New("fail")
		metricAdapter.PostMetricError = expectedErr

		metric := &stackdriver.Metric{}
		subject.PostMetric(metric)

		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(1))

		var err error
		Eventually(errs).Should(Receive(&err))
		Expect(err).To(Equal(expectedErr))
	})

	It("posts current batch when encounters a duplicate", func() {
		subject, _ := stackdriver.NewMetricsBuffer(5, metricAdapter)

		subject.PostMetric(&stackdriver.Metric{Name: "a", Value: 1})
		subject.PostMetric(&stackdriver.Metric{Name: "b", Value: 2})
		subject.PostMetric(&stackdriver.Metric{Name: "a", Value: 2})
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(2))

		Expect(metricAdapter.GetPostedMetrics()).To(Equal([]stackdriver.Metric{
			{Name: "a", Value: 1},
			{Name: "b", Value: 2},
		}))

		subject.PostMetric(&stackdriver.Metric{Name: "b", Value: 3})
		subject.PostMetric(&stackdriver.Metric{Name: "c"})
		subject.PostMetric(&stackdriver.Metric{Name: "d"})
		subject.PostMetric(&stackdriver.Metric{Name: "e"})
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(7))

		Expect(metricAdapter.GetPostedMetrics()).To(Equal([]stackdriver.Metric{
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
