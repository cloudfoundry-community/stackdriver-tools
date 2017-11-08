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
	"errors"
	"time"

	"sync"

	"github.com/cloudfoundry/lager"
)

var CounterStoppedErr = errors.New("attempted to increment counter without starting counter, further attempts will not be reported")

const Action = "heartbeater"

type Counter interface {
	Start()
	Increment(name string)
	IncrementBy(name string, count int)
	Stop()
}

type counter struct {
	logger lager.Logger
	period time.Duration

	started             bool
	done                chan struct{}
	nonStarterErrorOnce sync.Once

	sink Sink

	mutex    sync.Mutex
	counters map[string]int
}

func NewCollector(logger lager.Logger, period time.Duration, handler Sink) Counter {
	return &counter{
		logger:   logger,
		period:   period,
		done:     make(chan struct{}),
		counters: make(map[string]int),
		sink:     handler}
}

func (t *counter) Start() {
	t.mutex.Lock()
	if t.started {
		t.logger.Fatal(Action, errors.New("attempting to start an already running counter"))
		return
	}
	t.started = true
	t.mutex.Unlock()

	trigger := time.NewTicker(t.period)
	go func() {
		for {
			select {
			case <-trigger.C:
				t.emit()
			case <-t.done:
				t.logger.Info(Action, lager.Data{"debug": "done"})
				return
			}
		}
	}()
}

func (t *counter) emit() {
	t.mutex.Lock()
	counters := t.counters
	t.counters = make(map[string]int)
	t.mutex.Unlock()

	t.logger.Info(Action, lager.Data{"counters": counters})

	if t.sink != nil {
		t.sink.Record(counters)
	}
}

func (t *counter) Increment(name string) {
	t.IncrementBy(name, 1)
}

func (t *counter) IncrementBy(name string, count int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.started {
		t.counters[name] += count
	} else {
		t.nonStarterErrorOnce.Do(func() {
			t.logger.Error(Action, CounterStoppedErr)
		})
	}
}

func (t *counter) Stop() {
	t.logger.Info(Action, lager.Data{"debug": "Stopping"})

	t.mutex.Lock()
	close(t.done)
	t.started = false
	t.mutex.Unlock()

	t.emit()
}
