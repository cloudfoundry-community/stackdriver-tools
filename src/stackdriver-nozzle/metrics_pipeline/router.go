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

func (r *router) PostMetrics(metrics []messages.Metric) error {
	for _, metric := range metrics {
		if r.metricEvents[metric.Type] {
			// TODO: seems strange to re-package this as a new slice
			r.metricAdapter.PostMetrics([]messages.Metric{metric})
		}

		if r.logEvents[metric.Type] {
			log := &messages.Log{
				Labels:  metric.Labels,
				Payload: metric,
			}
			r.logAdapter.PostLog(log)
		}
	}

	return nil
}
