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
	"time"

	"golang.org/x/net/context"
)

// Reporter aggregates telemetry registered with expvar and reports it to Sink endpoints
type Reporter interface {
	Start(ctx context.Context)
}

type reporter struct {
	period time.Duration
	sinks  []Sink
}

// NewReporter provides a time based Reporter
func NewReporter(period time.Duration, sinks ...Sink) Reporter {
	return &reporter{period: period, sinks: sinks}
}

func (r *reporter) Start(ctx context.Context) {
	ticker := time.NewTicker(r.period)

	data := r.data()
	for _, sink := range r.sinks {
		sink.Init(data)
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				r.report()
			case <-ctx.Done():
				r.report()
				return
			}
		}
	}()
}

func (r *reporter) report() {
	data := r.data()
	for _, sink := range r.sinks {
		sink.Report(data)
	}
}

func (r *reporter) data() []*expvar.KeyValue {
	points := []*expvar.KeyValue{}

	forEachMetric(func(point expvar.KeyValue) {
		points = append(points, &point)
	})

	return points
}
