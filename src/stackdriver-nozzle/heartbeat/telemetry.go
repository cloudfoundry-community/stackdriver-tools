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

package heartbeat

import (
	"errors"
	"time"

	"sync"

	"github.com/cloudfoundry/lager"
)

type telemetry struct {
	logger lager.Logger
	period time.Duration

	started             bool
	done                chan struct{}
	nonStarterErrorOnce sync.Once

	handler Handler

	mutex    sync.Mutex
	counters map[string]uint
}

func NewTelemetry(logger lager.Logger, period time.Duration, handler Handler) Heartbeater {
	return &telemetry{
		logger:   logger,
		period:   period,
		done:     make(chan struct{}),
		counters: make(map[string]uint),
		handler:  handler}
}

func (t *telemetry) Start() {
	t.mutex.Lock()
	if t.started {
		t.logger.Fatal("heartbeater", errors.New("attempting to start an already running heartbeater"))
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
				t.logger.Info("heartbeater", lager.Data{"debug": "done"})
				return
			}
		}
	}()
}

func (t *telemetry) emit() {
	t.mutex.Lock()
	counters := t.counters
	t.counters = make(map[string]uint)
	t.mutex.Unlock()

	t.logger.Info("heartbeater", lager.Data{"counters": counters})

	if t.handler == nil {
		return
	}

	for name, count := range counters {
		t.handler.Handle(name, count)
	}

	t.handler.Flush()
}

func (t *telemetry) Increment(name string) {
	t.IncrementBy(name, 1)
}

func (t *telemetry) IncrementBy(name string, count uint) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.started {
		t.counters[name] += count
	} else {
		t.nonStarterErrorOnce.Do(func() {
			t.logger.Error("heartbeater", HeartbeaterStoppedErr)
		})
	}
}

func (t *telemetry) Stop() {
	t.logger.Info("heartbeater", lager.Data{"debug": "Stopping"})

	t.mutex.Lock()
	close(t.done)
	t.started = false
	t.mutex.Unlock()

	t.emit()
}
