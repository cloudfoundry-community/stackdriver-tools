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
	"context"
	"expvar"
	"math"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
)

const maxExpirePeriod = 10 * time.Second

var countersExpiredCount *telemetry.Counter

func init() {
	countersExpiredCount = telemetry.NewCounter(telemetry.Nozzle, "metrics.counters.expired")
}

type counterData struct {
	startTime     time.Time
	totalValue    *expvar.Int
	lastValue     uint64
	lastSeenTime  time.Time
	lastEventTime time.Time
}

// CounterTracker is used to provide a "start time" for each loggregator counter metric exported by the nozzle.
//
// Stackdriver requires each point for a cumulative metric to include "start time" in addition to the actual event time
// (aka "end time"): https://cloud.google.com/monitoring/api/ref_v3/rest/v3/TimeSeries#point
// Typically start time would correspond to the time when the actual process exporting the metric started. This ensures
// that when a process is restarted (and counter gets reset to 0), start time increases.
//
// Since binaries that export counter events to loggregator only provide event time, the nozzle needs to determine start
// time for each metric itself. To do that, CounterTracker keeps its own counter for each metric, which corresponds to the
// total number of events since the metric was first seen by the nozzle (which is exported as the start time).
//
// As an example, a series of incoming CounterEvents with total values of [100, 110, 115, 150] will be exported by the
// nozzle as [10, 15, 50] (first point seen by the nozzle is discarded, because each point reported to Stackdriver needs
// to cover non-zero time interval between start time and end time).
//
// If CounterTracker detects the total value for a given counter decrease, it will interpret this as a counter reset. This
// will not result in the Stackdriver cumulative metric being reset as well; for example, incoming CounterEvents with total
// values of [100, 110, 115, 10, 17] will be exported by the nozzle as [10, 15, 25, 32].
//
// CounterTracker will regularly remove internal state for metrics that have not been seen for a while. This is done to
// conserve memory, and also to ensure that old values do not re-surface if a given counter stops being exported for some
// period of time.
type CounterTracker struct {
	counters map[string]*counterData
	mu       *sync.Mutex // protects `counters`
	ttl      time.Duration
	logger   lager.Logger
	ticker   *time.Ticker
	ctx      context.Context
}

// NewCounterTracker creates and returns a counter tracker.
func NewCounterTracker(ctx context.Context, ttl time.Duration, logger lager.Logger) *CounterTracker {
	expirePeriod := time.Duration(ttl.Nanoseconds() / 2)
	if expirePeriod > maxExpirePeriod {
		expirePeriod = maxExpirePeriod
	}
	c := &CounterTracker{
		counters: map[string]*counterData{},
		mu:       &sync.Mutex{},
		ttl:      ttl,
		logger:   logger,
		ticker:   time.NewTicker(expirePeriod),
		ctx:      ctx,
	}
	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.expire()
			case <-c.ctx.Done():
				c.ticker.Stop()
				return
			}
		}
	}()
	return c
}

// Update accepts a counter name, event time and a value, and returns the total value for the counter along with its
// start time. Counter name provided needs to uniquely identify the time series (so it needs to include metric name as
// well as all metric label values).
// At least two values need to be observed for a given counter to determine the total value, so for the first observed
// value, 0 will be returned as the total, and end time will be equal to event time. Such points should not be reported
// to Stackdriver, since it expects points covering non-zero time interval.
func (t *CounterTracker) Update(name string, value uint64, eventTime time.Time) (int64, time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, present := t.counters[name]
	if !present {
		c = t.newCounterData(name, eventTime)
		t.counters[name] = c
	} else {
		var delta uint64
		if c.lastValue > value {
			// Counter has been reset.
			delta = value
		} else {
			delta = value - c.lastValue
		}
		if uint64(c.totalValue.Value())+delta > math.MaxInt64 {
			// Accumulated value overflows int64, we need to reset the counter.
			c.totalValue.Set(int64(delta))
			c.startTime = c.lastEventTime
		} else {
			c.totalValue.Add(int64(delta))
		}
	}
	c.lastValue = value
	c.lastSeenTime = time.Now()
	c.lastEventTime = eventTime
	return c.totalValue.Value(), c.startTime
}

func (t *CounterTracker) newCounterData(name string, eventTime time.Time) *counterData {
	var v *expvar.Int
	existing := expvar.Get(name)
	if existing != nil {
		// There was a previous counter with this name; use it instead, but reset value to 0.
		v = existing.(*expvar.Int)
		v.Set(0)
	} else {
		v = expvar.NewInt(name)
	}
	// Initialize counter state for a new counter.
	return &counterData{
		totalValue: v,
		startTime:  eventTime,
	}
}

func (t *CounterTracker) expire() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for name, counter := range t.counters {
		if time.Since(counter.lastSeenTime) > t.ttl {
			t.logger.Info("CounterTracker", lager.Data{
				"info":    "removing expired counter",
				"name":    name,
				"counter": counter,
				"value":   t.counters[name].totalValue.Value(),
			})
			// Reset values to -1 to make expired counters visible in /debug/vars.
			t.counters[name].totalValue.Set(-1)
			delete(t.counters, name)
			countersExpiredCount.Increment()
		}
	}
}
