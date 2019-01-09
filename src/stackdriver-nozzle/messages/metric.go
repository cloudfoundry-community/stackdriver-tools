package messages

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/genproto/googleapis/api/label"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/monitoring/v3"
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

func (m *Metric) metricKind() metric.MetricDescriptor_MetricKind {
	if m.IsCumulative() {
		return metric.MetricDescriptor_CUMULATIVE
	}
	return metric.MetricDescriptor_GAUGE
}

func (m *Metric) valueType() metric.MetricDescriptor_ValueType {
	if m.IsCumulative() {
		return metric.MetricDescriptor_INT64
	}
	return metric.MetricDescriptor_DOUBLE
}

// NeedsMetricDescriptor determines whether a custom metric descriptor needs to be created for this metric in Stackdriver.
// We do that if we need to set a custom unit, or mark metric as a cumulative.
func (m *Metric) NeedsMetricDescriptor() bool {
	return m.Unit != "" || m.IsCumulative()
}

// MetricDescriptor returns a Stackdriver MetricDescriptor proto for this metric.
func (m *Metric) MetricDescriptor(projectName string) *metric.MetricDescriptor {
	metricType := m.metricType()

	var labelDescriptors []*label.LabelDescriptor
	for key := range m.Labels {
		labelDescriptors = append(labelDescriptors, &label.LabelDescriptor{
			Key:       key,
			ValueType: label.LabelDescriptor_STRING,
		})
	}

	return &metric.MetricDescriptor{
		Name:        path.Join(projectName, "metricDescriptors", metricType),
		Type:        metricType,
		Labels:      labelDescriptors,
		MetricKind:  m.metricKind(),
		ValueType:   m.valueType(),
		Unit:        m.Unit,
		Description: "stackdriver-nozzle created custom metric.",
		DisplayName: m.Name,
	}
}

// TimeSeries returns a Stackdriver TimeSeries proto for this metric value.
func (m *Metric) TimeSeries() *monitoring.TimeSeries {
	var value *monitoring.TypedValue
	if m.IsCumulative() {
		value = &monitoring.TypedValue{Value: &monitoring.TypedValue_Int64Value{Int64Value: m.IntValue}}
	} else {
		value = &monitoring.TypedValue{Value: &monitoring.TypedValue_DoubleValue{DoubleValue: m.Value}}
	}

	point := &monitoring.Point{
		Interval: &monitoring.TimeInterval{
			EndTime:   &timestamp.Timestamp{Seconds: m.EventTime.Unix(), Nanos: int32(m.EventTime.Nanosecond())},
			StartTime: &timestamp.Timestamp{Seconds: m.StartTime.Unix(), Nanos: int32(m.StartTime.Nanosecond())},
		},
		Value: value,
	}
	return &monitoring.TimeSeries{
		MetricKind: m.metricKind(),
		ValueType:  m.valueType(),
		Metric: &metric.Metric{
			Type:   m.metricType(),
			Labels: m.Labels,
		},
		Points: []*monitoring.Point{point},
	}
}

func (m *Metric) Hash() string {
	var b bytes.Buffer

	b.Write([]byte(m.Name))
	if len(m.Labels) > 0 {
		b.WriteByte(',')
		b.WriteString(Flatten(m.Labels))
	}
	return b.String()
}

// Flatten serializes a set of label keys and values.
func Flatten(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf := &bytes.Buffer{}
	for i, k := range keys {
		buf.WriteString(fmt.Sprintf("%q=%q", k, labels[k]))
		if i+1 < len(keys) {
			buf.WriteByte(',')
		}
	}
	return buf.String()
}
