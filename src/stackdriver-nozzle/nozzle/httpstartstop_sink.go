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
	"fmt"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/sonde-go/events"
)

const httpPrefix telemetry.MetricPrefix = "app-http"

var (
	requestCount *telemetry.CounterMap
	responseCode *telemetry.CounterMap

	defaultLabels = []string{"job", "index", "applicationPath", "instanceIndex"}
)

func init() {
	requestCount = telemetry.NewCounterMap(httpPrefix, "request_count",
		defaultLabels...)
	responseCode = telemetry.NewCounterMap(httpPrefix, "response_code",
		append(defaultLabels, "code")...)
}

type httpSink struct {
	logger     lager.Logger
	labelMaker LabelMaker
}

// NewHttpSink returns a Sink that can receive sonde HttpStartStop events
// and generate per-application HTTP metrics from them.
func NewHttpSink(logger lager.Logger, labelMaker LabelMaker) Sink {
	return &httpSink{
		logger:     logger,
		labelMaker: labelMaker,
	}
}

func (sink *httpSink) Receive(envelope *events.Envelope) {
	if envelope.GetEventType() != events.Envelope_HttpStartStop {
		return
	}

	labels := sink.labelMaker.MetricLabels(envelope, false)
	if labels["applicationPath"] == "" {
		// We're only interested in HTTP traffic passing through the gorouters
		// to known applications.
		return
	}
	labelValues := defaultLabelValues(labels)

	if rcc, err := requestCount.Counter(labelValues...); err == nil {
		rcc.Increment()
	} else {
		sink.logger.Error("httpSink.Receive", fmt.Errorf("incrementing requestCount: %v", err))
	}
	code := fmt.Sprintf("%d", envelope.GetHttpStartStop().GetStatusCode())
	if rcc, err := responseCode.Counter(append(labelValues, code)...); err == nil {
		rcc.Increment()
	} else {
		sink.logger.Error("httpSink.Receive", fmt.Errorf("incrementing responseCode: %v", err))
	}
}

func defaultLabelValues(labels map[string]string) []string {
	labelValues := make([]string, len(defaultLabels))
	for i, key := range defaultLabels {
		labelValues[i] = labels[key]
	}
	return labelValues
}
