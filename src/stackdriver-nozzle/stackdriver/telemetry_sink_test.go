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
		oldData := &expvar.KeyValue{Key: "old", Value: &telemetry.Counter{}}
		newData := &expvar.KeyValue{Key: "new", Value: &telemetry.Counter{}}

		BeforeEach(func() {
			client.ListMetricDescriptorFn = func(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
				return []*metricpb.MetricDescriptor{
					{Name: projectPath + "/metricDescriptors/custom.googleapis.com/stackdriver-nozzle/" + oldData.Key},
				}, nil
			}

			sink.Init([]*expvar.KeyValue{oldData, newData})
		})

		It("only creates new metric descriptors", func() {
			Expect(client.DescriptorReqs).To(HaveLen(1))

			req := client.DescriptorReqs[0]
			Expect(req.Name).To(Equal(projectPath))
			descriptor := req.MetricDescriptor

			displayName := telemetry.Nozzle.Qualify(newData.Key)
			metricType := "custom.googleapis.com/" + displayName
			name := projectPath + "/metricDescriptors/" + metricType

			Expect(descriptor.Name).To(Equal(name))
			Expect(descriptor.Type).To(Equal(metricType))
			Expect(descriptor.DisplayName).To(Equal(displayName))
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
			Expect(metric.Type).To(Equal("custom.googleapis.com/stackdriver-nozzle/" + keyValue.Key))

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
		value := &telemetry.CounterMap{}
		mapVar := &expvar.KeyValue{Key: "earth", Value: value}
		BeforeEach(func() {
			oceanValue := &telemetry.Counter{}
			continentValue := &telemetry.Counter{}
			oceanValue.Set(5)
			continentValue.Set(7)
			value.Set("oceans", oceanValue)
			value.Set("continents", continentValue)
		})
		It("Init creates MetricDescriptors with label", func() {
			sink.Init([]*expvar.KeyValue{mapVar})

			Expect(client.DescriptorReqs).To(HaveLen(1))
			req := client.DescriptorReqs[0]
			labels := req.MetricDescriptor.Labels
			Expect(labels).To(HaveLen(3))
			Expect(labels).To(ContainElement(&labelpb.LabelDescriptor{Key: value.Category(), ValueType: labelpb.LabelDescriptor_STRING}))
		})

		It("Report posts TimeSeries with label", func() {
			sink.Report([]*expvar.KeyValue{mapVar})

			Expect(client.MetricReqs).To(HaveLen(1))
			req := client.MetricReqs[0]
			Expect(req.TimeSeries).To(HaveLen(2))
			kinds := map[string]*monitoringpb.TimeSeries{}
			for _, series := range req.TimeSeries {
				kinds[series.Metric.Labels[value.Category()]] = series
			}
			Expect(kinds).To(HaveKey("oceans"))
			Expect(kinds).To(HaveKey("continents"))
			Expect(kinds["oceans"].Points[0].Value.Value.(*monitoringpb.TypedValue_Int64Value).Int64Value).To(Equal(int64(5)))
			Expect(kinds["continents"].Points[0].Value.Value.(*monitoringpb.TypedValue_Int64Value).Int64Value).To(Equal(int64(7)))
		})
	})
})
