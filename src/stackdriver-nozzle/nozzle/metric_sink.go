/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nozzle

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
)

// NewLogSink returns a Sink that can receive sonde Events, translate them and send them to a stackdriver.MetricAdapter
func NewMetricSink(logger lager.Logger, pathPrefix string, labelMaker LabelMaker, metricAdapter stackdriver.MetricAdapter, ct *CounterTracker, unitParser UnitParser, runtimeMetricRegex string) (Sink, error) {
	r, err := regexp.Compile(runtimeMetricRegex)
	if err != nil {
		return nil, fmt.Errorf("cannot compile runtime metric regex: %v", err)
	}
	return &metricSink{
		pathPrefix:      pathPrefix,
		labelMaker:      labelMaker,
		metricAdapter:   metricAdapter,
		unitParser:      unitParser,
		counterTracker:  ct,
		logger:          logger,
		runtimeMetricRe: r,
	}, nil
}

type metricSink struct {
	pathPrefix      string
	labelMaker      LabelMaker
	metricAdapter   stackdriver.MetricAdapter
	unitParser      UnitParser
	counterTracker  *CounterTracker
	logger          lager.Logger
	runtimeMetricRe *regexp.Regexp
}

// isRuntimeMetric determines whether a given metric is a runtime metric.
// "Runtime metrics" are the ones that are exported by multiple processes (with different values of the 'origin' label).
// By default 'origin' label value gets prepended to metric name, however for runtime metrics we instead add it as a metric label.
// As the result, instead of creating a separate copy of each runtime metric per origin, we have a single metric with origin available as a label.
// This allows aggregating values of these metrics across origins, and also helps us stay below the Stackdriver limit for the number of custom metrics.
func (ms *metricSink) isRuntimeMetric(envelope *events.Envelope) bool {
	return envelope.GetEventType() == events.Envelope_ValueMetric && ms.runtimeMetricRe.MatchString(envelope.GetValueMetric().GetName())
}

func (ms *metricSink) getPrefix(envelope *events.Envelope) string {
	buf := bytes.Buffer{}
	if ms.pathPrefix != "" {
		buf.WriteString(ms.pathPrefix)
		buf.WriteString("/")
	}
	// Non-runtime metrics get origin prepended to metric name.
	if !ms.isRuntimeMetric(envelope) && envelope.GetOrigin() != "" {
		buf.WriteString(envelope.GetOrigin())
		buf.WriteString(".")
	}
	return buf.String()
}

func (ms *metricSink) Receive(envelope *events.Envelope) {
	labels := ms.labelMaker.MetricLabels(envelope, ms.isRuntimeMetric(envelope))
	metricPrefix := ms.getPrefix(envelope)
	eventType := envelope.GetEventType()

	timestamp := time.Duration(envelope.GetTimestamp())
	eventTime := time.Unix(
		int64(timestamp/time.Second),
		int64(timestamp%time.Second),
	)

	var metrics []*messages.Metric
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric:
		valueMetric := envelope.GetValueMetric()
		metrics = []*messages.Metric{{
			Name:      metricPrefix + valueMetric.GetName(),
			Labels:    labels,
			Type:      eventType,
			Value:     valueMetric.GetValue(),
			EventTime: eventTime,
			StartTime: eventTime,
			Unit:      ms.unitParser.Parse(valueMetric.GetUnit()),
		}}
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		metrics = []*messages.Metric{
			{Name: metricPrefix + "diskBytesQuota", Value: float64(containerMetric.GetDiskBytesQuota())},
			{Name: metricPrefix + "cpuPercentage", Value: float64(containerMetric.GetCpuPercentage())},
			{Name: metricPrefix + "diskBytes", Value: float64(containerMetric.GetDiskBytes())},
			{Name: metricPrefix + "memoryBytes", Value: float64(containerMetric.GetMemoryBytes())},
			{Name: metricPrefix + "memoryBytesQuota", Value: float64(containerMetric.GetMemoryBytesQuota())},
		}
		for _, metric := range metrics {
			metric.Labels = labels
			metric.Type = eventType
			metric.EventTime = eventTime
			metric.StartTime = eventTime
		}
	case events.Envelope_CounterEvent:
		counterEvent := envelope.GetCounterEvent()
		if ms.counterTracker == nil {
			// When there is no counter tracker, report CounterEvent metrics as two gauges: 'delta' and 'total'.
			metrics = []*messages.Metric{
				{
					Name:      fmt.Sprintf("%s%v.delta", metricPrefix, counterEvent.GetName()),
					Labels:    labels,
					Type:      events.Envelope_ValueMetric,
					Value:     float64(counterEvent.GetDelta()),
					EventTime: eventTime,
					StartTime: eventTime,
				},
				{
					Name:      fmt.Sprintf("%s%v.total", metricPrefix, counterEvent.GetName()),
					Labels:    labels,
					Type:      events.Envelope_ValueMetric,
					Value:     float64(counterEvent.GetTotal()),
					EventTime: eventTime,
					StartTime: eventTime,
				},
			}
		} else {
			// Create a partial metric struct (lacking IntValue and StartTime) to allow determining metric.Hash (used as
			// the counter name) based on metric name and labels.
			metric := &messages.Metric{
				Name:      metricPrefix + counterEvent.GetName(),
				Labels:    labels,
				Type:      eventType,
				EventTime: eventTime,
			}
			total, st := ms.counterTracker.Update(metric.Hash(), counterEvent.GetTotal(), eventTime)
			// Stackdriver expects non-zero time intervals, so only add a metric if event time is older than start time.
			if eventTime.After(st) {
				metric.StartTime = st
				metric.IntValue = total
				metrics = append(metrics, metric)
			}
		}
	default:
		ms.logger.Error("metricSink.Receive", fmt.Errorf("unknown event type: %v", envelope.EventType))
		return
	}

	ms.metricAdapter.PostMetrics(metrics)
}
