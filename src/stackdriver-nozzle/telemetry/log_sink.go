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

package telemetry

import (
	"expvar"

	"github.com/cloudfoundry/lager"
)

// NewLogSink provides a Sink that writes Reports to a lager.Logger
func NewLogSink(logger lager.Logger) Sink {
	return &logSink{logger, make(map[string]int64)}
}

type logSink struct {
	logger lager.Logger

	lastReport map[string]int64
}

func (ls *logSink) Init([]*expvar.KeyValue) {
	ls.logger.Info("heartbeater", lager.Data{"info": "started"})
}

func (ls *logSink) Report(values []*expvar.KeyValue) {
	report := map[string]int64{}
	reportDelta := map[string]int64{}

	for _, val := range values {
		if intVal, ok := val.Value.(*expvar.Int); ok {
			key := val.Key
			report[key] = intVal.Value()
			reportDelta[key] = report[key] - ls.lastReport[key]
		}
	}

	ls.lastReport = report
	ls.logger.Info("heartbeater", lager.Data{"counters.cumulative": report, "counters.delta": reportDelta})
}
