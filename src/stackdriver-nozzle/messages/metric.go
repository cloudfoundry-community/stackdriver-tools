package messages

import (
	"bytes"
	"sort"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
)

type Metric struct {
	Name      string
	Type      events.Envelope_EventType
	Value     float64
	Labels    map[string]string
	EventTime time.Time
	Unit      string // TODO Should this be "1" if it's empty?
}

func (m *Metric) Hash() string {
	var b bytes.Buffer
	b.Write([]byte(m.Name))

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
