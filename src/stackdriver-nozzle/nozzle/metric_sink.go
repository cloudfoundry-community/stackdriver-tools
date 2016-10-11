package nozzle

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

func NewMetricSink(labelMaker LabelMaker, metricAdapter stackdriver.MetricAdapter, unitParser UnitParser) Sink {
	return &metricSink{
		labelMaker:    labelMaker,
		metricAdapter: metricAdapter,
		unitParser:    unitParser,
	}
}

type metricSink struct {
	labelMaker    LabelMaker
	metricAdapter stackdriver.MetricAdapter
	unitParser    UnitParser
}

func (ms *metricSink) Receive(envelope *events.Envelope) error {
	labels := ms.labelMaker.Build(envelope)

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
			Unit:      ms.unitParser.Parse(valueMetric.GetUnit()),
		}}
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		metrics = []stackdriver.Metric{
			{Name: "diskBytesQuota", Value: float64(containerMetric.GetDiskBytesQuota()), Labels: labels, EventTime: eventTime},
			{Name: "instanceIndex", Value: float64(containerMetric.GetInstanceIndex()), Labels: labels, EventTime: eventTime},
			{Name: "cpuPercentage", Value: float64(containerMetric.GetCpuPercentage()), Labels: labels, EventTime: eventTime},
			{Name: "diskBytes", Value: float64(containerMetric.GetDiskBytes()), Labels: labels, EventTime: eventTime},
			{Name: "memoryBytes", Value: float64(containerMetric.GetMemoryBytes()), Labels: labels, EventTime: eventTime},
			{Name: "memoryBytesQuota", Value: float64(containerMetric.GetMemoryBytesQuota()), Labels: labels, EventTime: eventTime},
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

	return ms.metricAdapter.PostMetrics(metrics)
}
