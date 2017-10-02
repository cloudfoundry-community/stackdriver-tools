package messages

import (
	"bytes"
	"sort"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
)

type Metric struct {
	Name      string
	Value     float64
	EventTime time.Time
	Unit      string // TODO Should this be "1" if it's empty?
}

// MetricEvent represents the translation of an events.Envelope into a set
// of Metrics
type MetricEvent struct {
	Labels  map[string]string `json:"-"`
	Metrics []*Metric
	Type    events.Envelope_EventType `json:"-"`
}

func (m *MetricEvent) Hash() string {
	var b bytes.Buffer

	// Extract keys to a slice and sort it
	keys := make([]string, len(m.Labels), len(m.Labels))
	for k, _ := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.Write([]byte(k))
		b.Write([]byte(m.Labels[k]))
	}
	return b.String()
}
