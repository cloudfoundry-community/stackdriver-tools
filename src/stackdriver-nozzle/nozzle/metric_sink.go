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
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
)

// NewLogSink returns a Sink that can receive sonde Events, translate them and send them to a stackdriver.MetricAdapter
func NewMetricSink(logger lager.Logger, pathPrefix string, labelMaker LabelMaker, metricAdapter stackdriver.MetricAdapter, unitParser UnitParser) Sink {
	return &metricSink{
		pathPrefix:    pathPrefix,
		labelMaker:    labelMaker,
		metricAdapter: metricAdapter,
		unitParser:    unitParser,
		logger:        logger,
	}
}

type metricSink struct {
	pathPrefix    string
	labelMaker    LabelMaker
	metricAdapter stackdriver.MetricAdapter
	unitParser    UnitParser
	logger        lager.Logger
}

func (ms *metricSink) getPrefix(envelope *events.Envelope) string {
	buf := bytes.Buffer{}
	if ms.pathPrefix != "" {
		buf.WriteString(ms.pathPrefix)
		buf.WriteString("/")
	}
	if envelope.GetOrigin() != "" {
		buf.WriteString(envelope.GetOrigin())
		buf.WriteString(".")
	}
	return buf.String()
}

func (ms *metricSink) Receive(envelope *events.Envelope) {
	labels := ms.labelMaker.MetricLabels(envelope)
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
			Unit:      ms.unitParser.Parse(valueMetric.GetUnit()),
		}}
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		metrics = []*messages.Metric{
			{Name: metricPrefix + "diskBytesQuota", Labels: labels, Type: eventType, Value: float64(containerMetric.GetDiskBytesQuota()), EventTime: eventTime},
			{Name: metricPrefix + "cpuPercentage", Labels: labels, Type: eventType, Value: float64(containerMetric.GetCpuPercentage()), EventTime: eventTime},
			{Name: metricPrefix + "diskBytes", Labels: labels, Type: eventType, Value: float64(containerMetric.GetDiskBytes()), EventTime: eventTime},
			{Name: metricPrefix + "memoryBytes", Labels: labels, Type: eventType, Value: float64(containerMetric.GetMemoryBytes()), EventTime: eventTime},
			{Name: metricPrefix + "memoryBytesQuota", Labels: labels, Type: eventType, Value: float64(containerMetric.GetMemoryBytesQuota()), EventTime: eventTime},
		}
	case events.Envelope_CounterEvent:
		counterEvent := envelope.GetCounterEvent()
		metrics = []*messages.Metric{
			{
				Name:      fmt.Sprintf("%s%v.delta", metricPrefix, counterEvent.GetName()),
				Labels:    labels,
				Type:      eventType,
				Value:     float64(counterEvent.GetDelta()),
				EventTime: eventTime,
			},
			{
				Name:      fmt.Sprintf("%s%v.total", metricPrefix, counterEvent.GetName()),
				Labels:    labels,
				Type:      eventType,
				Value:     float64(counterEvent.GetTotal()),
				EventTime: eventTime,
			},
		}
	default:
		ms.logger.Error("metricSink.Receive", fmt.Errorf("unknown event type: %v", envelope.EventType))
		return
	}

	ms.metricAdapter.PostMetrics(metrics)
}
