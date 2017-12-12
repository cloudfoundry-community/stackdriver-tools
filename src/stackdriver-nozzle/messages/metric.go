package messages

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/golang/protobuf/ptypes/timestamp"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

// Metric represents one of the metrics contained in an events.Envelope.
type Metric struct {
	Name      string
	Labels    map[string]string `json:"-"`
	Value     float64
	IntValue  int64
	EventTime time.Time
	StartTime time.Time                 `json:"-"`
	Unit      string                    // TODO Should this be "1" if it's empty?
	Type      events.Envelope_EventType `json:"-"`
}

func (m *Metric) IsCumulative() bool {
	return m.Type == events.Envelope_CounterEvent
}

func (m *Metric) metricType() string {
	return path.Join("custom.googleapis.com", m.Name)
}

// NeedsMetricDescriptor determines whether a custom metric descriptor needs to be created for this metric in Stackdriver.
// We do that if we need to set a custom unit, or mark metric as a cumulative.
func (m *Metric) NeedsMetricDescriptor() bool {
	return m.Unit != "" || m.IsCumulative()
}

// MetricDescriptor returns a Stackdriver MetricDescriptor proto for this metric.
func (m *Metric) MetricDescriptor(projectName string) *metricpb.MetricDescriptor {
	metricType := m.metricType()

	var labelDescriptors []*labelpb.LabelDescriptor
	for key := range m.Labels {
		labelDescriptors = append(labelDescriptors, &labelpb.LabelDescriptor{
			Key:       key,
			ValueType: labelpb.LabelDescriptor_STRING,
		})
	}

	metricKind := metricpb.MetricDescriptor_GAUGE
	valueType := metricpb.MetricDescriptor_DOUBLE
	if m.IsCumulative() {
		metricKind = metricpb.MetricDescriptor_CUMULATIVE
		valueType = metricpb.MetricDescriptor_INT64
	}

	return &metricpb.MetricDescriptor{
		Name:        path.Join(projectName, "metricDescriptors", metricType),
		Type:        metricType,
		Labels:      labelDescriptors,
		MetricKind:  metricKind,
		ValueType:   valueType,
		Unit:        m.Unit,
		Description: "stackdriver-nozzle created custom metric.",
		DisplayName: m.Name,
	}
}

// TimeSeries returns a Stackdriver TimeSeries proto for this metric value.
func (m *Metric) TimeSeries() *monitoringpb.TimeSeries {
	var value *monitoringpb.TypedValue
	if m.IsCumulative() {
		value = &monitoringpb.TypedValue{Value: &monitoringpb.TypedValue_Int64Value{Int64Value: m.IntValue}}
	} else {
		value = &monitoringpb.TypedValue{Value: &monitoringpb.TypedValue_DoubleValue{DoubleValue: m.Value}}
	}

	point := &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &timestamp.Timestamp{Seconds: m.EventTime.Unix(), Nanos: int32(m.EventTime.Nanosecond())},
			StartTime: &timestamp.Timestamp{Seconds: m.StartTime.Unix(), Nanos: int32(m.StartTime.Nanosecond())},
		},
		Value: value,
	}
	return &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type:   m.metricType(),
			Labels: m.Labels,
		},
		Points: []*monitoringpb.Point{point},
	}
}

func (m *Metric) Hash() string {
	var b bytes.Buffer

	b.Write([]byte(m.Name))

	// Extract label keys to a slice and sort it
	keys := make([]string, 0, len(m.Labels))
	for k := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.Write([]byte(fmt.Sprintf(",%s='%s'", k, m.Labels[k])))
	}
	return b.String()
}
