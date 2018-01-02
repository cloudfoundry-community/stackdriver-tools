package stackdriver

import (
	"expvar"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"

	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const (
	projectID      = "myproject"
	projectPath    = "projects/" + projectID
	subscriptionID = "sdnozzle"
	foundation     = "bosh"
)

var _ = Describe("TelemetrySink", func() {
	var (
		logger *mocks.MockLogger
		sink   telemetry.Sink
		client *mocks.MockClient
	)
	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		client = &mocks.MockClient{}
		sink = NewTelemetrySink(logger, client, projectID, subscriptionID, foundation)
	})

	Context("Init with existing MetricDescriptors", func() {
		oldData := &expvar.KeyValue{Key: telemetry.Nozzle.Qualify("old"), Value: &telemetry.Counter{}}
		newData := &expvar.KeyValue{Key: telemetry.Nozzle.Qualify("new"), Value: &telemetry.Counter{}}

		BeforeEach(func() {
			client.ListMetricDescriptorFn = func(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
				return []*metricpb.MetricDescriptor{
					{Name: projectPath + "/metricDescriptors/custom.googleapis.com/" + oldData.Key},
				}, nil
			}

			sink.Init([]*expvar.KeyValue{oldData, newData})
		})

		It("only creates new metric descriptors", func() {
			Expect(client.DescriptorReqs).To(HaveLen(1))

			req := client.DescriptorReqs[0]
			Expect(req.Name).To(Equal(projectPath))
			descriptor := req.MetricDescriptor

			metricType := "custom.googleapis.com/" + newData.Key
			name := projectPath + "/metricDescriptors/" + metricType

			Expect(descriptor.Name).To(Equal(name))
			Expect(descriptor.Type).To(Equal(metricType))
			Expect(descriptor.DisplayName).To(Equal(newData.Key))
			Expect(descriptor.MetricKind).To(Equal(metricpb.MetricDescriptor_CUMULATIVE))
			Expect(descriptor.ValueType).To(Equal(metricpb.MetricDescriptor_INT64))

			labels := descriptor.Labels
			Expect(labels).To(HaveLen(2))
			Expect(labels).To(ContainElement(&labelpb.LabelDescriptor{Key: "foundation", ValueType: labelpb.LabelDescriptor_STRING}))
			Expect(labels).To(ContainElement(&labelpb.LabelDescriptor{Key: "subscription_id", ValueType: labelpb.LabelDescriptor_STRING}))
		})
	})

	Context("Report", func() {
		value := &telemetry.Counter{}
		keyValue := &expvar.KeyValue{Key: "foo", Value: value}
		BeforeEach(func() {
			value.Set(1234)
			sink.Report([]*expvar.KeyValue{keyValue})
		})

		It("posts TimeSeries", func() {
			Expect(client.MetricReqs).To(HaveLen(1))

			req := client.MetricReqs[0]
			Expect(req.Name).To(Equal(projectPath))
			Expect(req.TimeSeries).To(HaveLen(1))

			series := req.TimeSeries[0]
			Expect(series.Resource).NotTo(BeNil())

			metric := series.Metric
			Expect(metric.Type).To(Equal("custom.googleapis.com/" + keyValue.Key))

			labels := metric.Labels
			Expect(labels).To(HaveLen(2))
			Expect(labels).To(HaveKeyWithValue("foundation", foundation))
			Expect(labels).To(HaveKeyWithValue("subscription_id", subscriptionID))

			Expect(series.Points).To(HaveLen(1))
			point := series.Points[0]
			Expect(point.Value.Value.(*monitoringpb.TypedValue_Int64Value).Int64Value).To(Equal(value.Value()))
		})
	})

	Context("with many metrics", func() {
		values := []*expvar.KeyValue{}
		BeforeEach(func() {
			for i := 0; i < 300; i++ {
				value := &telemetry.Counter{}
				value.Set(int64(i))
				values = append(values, &expvar.KeyValue{Key: fmt.Sprintf("foo%d", i), Value: value})
			}

			sink.Report(values)
		})

		It("batches requests to Stackdriver", func() {
			Expect(client.MetricReqs).To(HaveLen(2))
			Expect(client.MetricReqs[0].TimeSeries).To(HaveLen(200))
			Expect(client.MetricReqs[1].TimeSeries).To(HaveLen(100))
		})
	})

	Context("with a Map", func() {
		value := &telemetry.CounterMap{LabelKeys: []string{"uuid", "code"}}
		mapVar := &expvar.KeyValue{Key: "response_code", Value: value}
		BeforeEach(func() {
			value.Init()
			firstOK := value.MustCounter("abcdef", "200")
			firstOK.Set(5)
			firstErr := value.MustCounter("abcdef", "500")
			firstErr.Set(4)
			secondOK := value.MustCounter("ghijkl", "200")
			secondOK.Set(3)
			secondErr := value.MustCounter("ghijkl", "500")
			secondErr.Set(2)
		})
		It("Init creates MetricDescriptors with label", func() {
			sink.Init([]*expvar.KeyValue{mapVar})

			Expect(client.DescriptorReqs).To(HaveLen(1))
			req := client.DescriptorReqs[0]
			labels := req.MetricDescriptor.Labels
			Expect(labels).To(HaveLen(4))
			Expect(labels).To(ContainElement(&labelpb.LabelDescriptor{Key: "uuid", ValueType: labelpb.LabelDescriptor_STRING}))
			Expect(labels).To(ContainElement(&labelpb.LabelDescriptor{Key: "code", ValueType: labelpb.LabelDescriptor_STRING}))
		})

		It("Report posts TimeSeries with label", func() {
			sink.Report([]*expvar.KeyValue{mapVar})

			Expect(client.MetricReqs).To(HaveLen(1))
			req := client.MetricReqs[0]
			Expect(req.TimeSeries).To(HaveLen(4))
			data := map[string]int64{}
			for _, series := range req.TimeSeries {
				key := fmt.Sprintf("%s.%s", series.Metric.Labels["uuid"], series.Metric.Labels["code"])
				value := series.Points[0].Value.Value.(*monitoringpb.TypedValue_Int64Value).Int64Value
				data[key] = value
			}
			Expect(data).To(Equal(map[string]int64{
				"abcdef.200": 5,
				"abcdef.500": 4,
				"ghijkl.200": 3,
				"ghijkl.500": 2,
			}))
		})
	})
})
