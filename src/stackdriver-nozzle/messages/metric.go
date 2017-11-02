package messages

import (
	"bytes"
	"sort"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
)

type DataPoint struct {
	Name  string
	Value float64
	Unit  string // TODO Should this be "1" if it's empty?
}

// MetricEvent represents the translation of an events.Envelope into DataPoints
type MetricEvent struct {
	Labels  map[string]string `json:"-"`
	Metrics []*DataPoint
	Time    time.Time
	Type    events.Envelope_EventType `json:"-"`
}

func (m *MetricEvent) Hash() string {
	var b bytes.Buffer

	// Extract keys to a slice and sort it
	numKeys := len(m.Metrics) + len(m.Labels)
	keys := make([]string, numKeys, numKeys)
	for _, m := range m.Metrics {
		keys = append(keys, m.Name)
	}
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
