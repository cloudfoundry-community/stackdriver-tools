package nozzle

import (
	"fmt"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	"time"
)

func NewMetricHandler(labelMaker LabelMaker, metricAdapter stackdriver.MetricAdapter) Handler {
	return &metricHandler{
		labelMaker:    labelMaker,
		metricAdapter: metricAdapter,
	}
}

type metricHandler struct {
	labelMaker    LabelMaker
	metricAdapter stackdriver.MetricAdapter
}

func (mh *metricHandler) HandleEnvelope(envelope *events.Envelope) error {
	labels := mh.labelMaker.Build(envelope)

	timestamp := time.Duration(envelope.GetTimestamp())
	eventTime := time.Unix(
		int64(timestamp/time.Second),
		int64(timestamp%time.Second),
	)

	var metrics []stackdriver.Metric
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric:
		valueMetric := envelope.GetValueMetric()
		metrics = []stackdriver.Metric{{
			Name:      valueMetric.GetName(),
			Value:     valueMetric.GetValue(),
			Labels:    labels,
			EventTime: eventTime,
		}}
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		metrics = []stackdriver.Metric{
			{"diskBytesQuota", float64(containerMetric.GetDiskBytesQuota()), labels, eventTime},
			{"instanceIndex", float64(containerMetric.GetInstanceIndex()), labels, eventTime},
			{"cpuPercentage", float64(containerMetric.GetCpuPercentage()), labels, eventTime},
			{"diskBytes", float64(containerMetric.GetDiskBytes()), labels, eventTime},
			{"memoryBytes", float64(containerMetric.GetMemoryBytes()), labels, eventTime},
			{"memoryBytesQuota", float64(containerMetric.GetMemoryBytesQuota()), labels, eventTime},
		}
	case events.Envelope_CounterEvent:
		counterEvent := envelope.GetCounterEvent()
		metrics = []stackdriver.Metric{{
			Name:      counterEvent.GetName(),
			Value:     float64(counterEvent.GetTotal()),
			Labels:    labels,
			EventTime: eventTime,
		}}
	default:
		return fmt.Errorf("unknown event type: %v", envelope.EventType)
	}

	return mh.metricAdapter.PostMetrics(metrics)
}
