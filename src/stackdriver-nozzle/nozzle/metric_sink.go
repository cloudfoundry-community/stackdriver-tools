package nozzle

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

func NewMetricSink(labelMaker LabelMaker, metricBuffer stackdriver.MetricsBuffer, unitParser UnitParser) Sink {
	return &metricSink{
		labelMaker:   labelMaker,
		metricBuffer: metricBuffer,
		unitParser:   unitParser,
	}
}

type metricSink struct {
	labelMaker   LabelMaker
	metricBuffer stackdriver.MetricsBuffer
	unitParser   UnitParser
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
			{Name: "diskBytesQuota", Value: float64(containerMetric.GetDiskBytesQuota()), EventTime: eventTime, Labels: labels},
			{Name: "instanceIndex", Value: float64(containerMetric.GetInstanceIndex()), EventTime: eventTime, Labels: labels},
			{Name: "cpuPercentage", Value: float64(containerMetric.GetCpuPercentage()), EventTime: eventTime, Labels: labels},
			{Name: "diskBytes", Value: float64(containerMetric.GetDiskBytes()), EventTime: eventTime, Labels: labels},
			{Name: "memoryBytes", Value: float64(containerMetric.GetMemoryBytes()), EventTime: eventTime, Labels: labels},
			{Name: "memoryBytesQuota", Value: float64(containerMetric.GetMemoryBytesQuota()), EventTime: eventTime, Labels: labels},
		}
	case events.Envelope_CounterEvent:
		counterEvent := envelope.GetCounterEvent()
		metrics = []stackdriver.Metric{{
			Name:      counterEvent.GetName(),
			Value:     float64(counterEvent.GetTotal()),
			EventTime: eventTime,
			Labels:    labels,
		}}
	default:
		return fmt.Errorf("unknown event type: %v", envelope.EventType)
	}

	for _, metric := range metrics {
		ms.metricBuffer.PostMetric(&metric)
	}
	return nil
}
