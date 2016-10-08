package stackdriver_test

import (
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

var _ = Describe("MetricAdapter", func() {
	var (
		subject stackdriver.MetricAdapter
		client  *mockClient
	)

	BeforeEach(func() {
		client = &mockClient{}
		subject = stackdriver.NewMetricAdapter("my-awesome-project", client)
	})

	It("takes metrics and posts a time series", func() {
		eventTime := time.Now()
		metrics := []stackdriver.Metric{
			{
				Name:  "metricName",
				Value: 123.45,
				Labels: map[string]string{
					"key": "name",
				},
				EventTime: eventTime,
			},
			{
				Name:  "secondMetricName",
				Value: 54.321,
				Labels: map[string]string{
					"secondKey": "secondName",
				},
				EventTime: eventTime,
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.reqs).To(HaveLen(1))

		req := client.reqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))

		timeSerieses := req.GetTimeSeries()
		Expect(timeSerieses).To(HaveLen(len(metrics)))

		timeSeries := timeSerieses[0]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/metricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(metrics[0].Labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point := timeSeries.GetPoints()[0]
		Expect(point.GetInterval().GetEndTime().Seconds).To(Equal(int64(eventTime.Second())))
		Expect(point.GetInterval().GetEndTime().Nanos).To(Equal(int32(eventTime.Nanosecond())))
		value, ok := point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(123.45))

		timeSeries = timeSerieses[1]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/secondMetricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(metrics[1].Labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point = timeSeries.GetPoints()[0]
		value, ok = point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(54.321))
	})
})

type mockClient struct {
	reqs []*monitoringpb.CreateTimeSeriesRequest
}

func (mc *mockClient) Post(req *monitoringpb.CreateTimeSeriesRequest) error {
	mc.reqs = append(mc.reqs, req)
	return nil
}
