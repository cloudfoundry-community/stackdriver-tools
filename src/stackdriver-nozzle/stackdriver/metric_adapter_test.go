package stackdriver_test

import (
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

var _ = Describe("MetricAdapter", func() {
	var (
		subject stackdriver.MetricAdapter
		client  *mockClient
	)

	BeforeEach(func() {
		client = &mockClient{}
		subject, _ = stackdriver.NewMetricAdapter("my-awesome-project", client)
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

		Expect(client.metricReqs).To(HaveLen(1))

		req := client.metricReqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))

		timeSerieses := req.GetTimeSeries()
		Expect(timeSerieses).To(HaveLen(len(metrics)))

		timeSeries := timeSerieses[0]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/metricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(metrics[0].Labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point := timeSeries.GetPoints()[0]
		Expect(point.GetInterval().GetEndTime().Seconds).To(Equal(int64(eventTime.Unix())))
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

	It("creates metric descriptors", func() {
		metrics := []stackdriver.Metric{
			{
				Name:   "metricWithUnit",
				Labels: map[string]string{"key": "value"},
				Unit:   "{foobar}",
			},
			{
				Name:   "metricWithoutUnit",
				Labels: map[string]string{"key": "value"},
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.descriptorReqs).To(HaveLen(1))
		req := client.descriptorReqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))
		Expect(req.MetricDescriptor).To(Equal(&metricpb.MetricDescriptor{
			Name:        "projects/my-awesome-project/metricDescriptors/custom.googleapis.com/metricWithUnit",
			Type:        "custom.googleapis.com/metricWithUnit",
			Labels:      []*labelpb.LabelDescriptor{{Key: "key", ValueType: 0, Description: ""}},
			MetricKind:  metricpb.MetricDescriptor_GAUGE,
			ValueType:   metricpb.MetricDescriptor_DOUBLE,
			Unit:        "{foobar}",
			Description: "stackdriver-nozzle created custom metric.",
			DisplayName: "metricWithUnit",
		}))
	})

	It("only creates the same descriptor once", func() {
		metrics := []stackdriver.Metric{
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "metricWithUnitToo",
				Unit: "{barfoo}",
			},
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "anExistingMetric",
				Unit: "{lalala}",
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.descriptorReqs).To(HaveLen(2))
	})
})

type mockClient struct {
	metricReqs     []*monitoringpb.CreateTimeSeriesRequest
	descriptorReqs []*monitoringpb.CreateMetricDescriptorRequest
}

func (mc *mockClient) Post(req *monitoringpb.CreateTimeSeriesRequest) error {
	mc.metricReqs = append(mc.metricReqs, req)
	return nil
}

func (mc *mockClient) CreateMetricDescriptor(request *monitoringpb.CreateMetricDescriptorRequest) error {
	mc.descriptorReqs = append(mc.descriptorReqs, request)
	return nil
}

func (mc *mockClient) ListMetricDescriptors(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
	return []*metricpb.MetricDescriptor{
		{Name: "anExistingMetric"},
	}, nil
}
