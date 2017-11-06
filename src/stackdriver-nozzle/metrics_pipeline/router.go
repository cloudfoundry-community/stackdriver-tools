package metrics_pipeline

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

type router struct {
	metricAdapter stackdriver.MetricAdapter
	logAdapter    stackdriver.LogAdapter
	logEvents     map[events.Envelope_EventType]bool
	metricEvents  map[events.Envelope_EventType]bool
}

// NewRouter provides a MetricAdapter that routes a given metric to
// Stackdriver Logging and Stackdriver Monitoring based on configuration
func NewRouter(metricAdapter stackdriver.MetricAdapter, metricEvents []events.Envelope_EventType, logAdapter stackdriver.LogAdapter, logEvents []events.Envelope_EventType) stackdriver.MetricAdapter {
	r := &router{metricAdapter: metricAdapter, logAdapter: logAdapter}

	r.metricEvents = make(map[events.Envelope_EventType]bool)
	for _, e := range metricEvents {
		r.metricEvents[e] = true
	}

	r.logEvents = make(map[events.Envelope_EventType]bool)
	for _, e := range logEvents {
		r.logEvents[e] = true
	}

	return r
}

func (r *router) PostMetricEvents(events []*messages.MetricEvent) {
	metricEvents := []*messages.MetricEvent{}
	for i := range events {
		if r.metricEvents[events[i].Type] {
			metricEvents = append(metricEvents, events[i])
		}

		if r.logEvents[events[i].Type] {
			log := &messages.Log{
				Labels:  events[i].Labels,
				Payload: events[i],
			}
			r.logAdapter.PostLog(log)
		}
	}

	if len(metricEvents) > 0 {
		r.metricAdapter.PostMetricEvents(metricEvents)
	}
}
