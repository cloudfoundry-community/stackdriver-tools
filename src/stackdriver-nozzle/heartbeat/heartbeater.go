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
	"fmt"
	"time"

	"github.com/cloudfoundry/lager"
)

type Heartbeater interface {
	Start()
	Increment(string)
	Stop()
}

type Handler interface {
	Handle(string)
	Flush() error
	Name() string
}

type heartbeater struct {
	logger   lager.Logger
	trigger  <-chan time.Time
	counter  chan string
	done     chan struct{}
	started  bool
	handlers []Handler
}

func NewHeartbeater(logger lager.Logger, trigger <-chan time.Time, prefix string) Heartbeater {
	counter := make(chan string)
	done := make(chan struct{})
	loggerHandler := NewLoggerHandler(logger, prefix)
	return &heartbeater{
		trigger: trigger,
		counter: counter,
		done:    done,
		started: false,
		logger:  logger,
		handlers: []Handler{
			loggerHandler,
		},
	}
}

func NewLoggerMetricHeartbeater(metricHandler Handler, logger lager.Logger, trigger <-chan time.Time, prefix string) Heartbeater {
	counter := make(chan string)
	done := make(chan struct{})
	loggerHandler := NewLoggerHandler(logger, prefix)
	return &heartbeater{
		trigger: trigger,
		counter: counter,
		done:    done,
		started: false,
		logger:  logger,
		handlers: []Handler{
			loggerHandler,
			metricHandler,
		},
	}
}
func (h *heartbeater) Start() {
	if h.started {
		h.logger.Error("heartbeater", errors.New("attempting to start an already running heartbeater"))
		return
	}

	h.logger.Info("heartbeater", lager.Data{"debug": "Starting heartbeater"})
	h.started = true
	go func() {
		for {
			select {
			case <-h.trigger:
				h.logger.Info("heartbeater", lager.Data{"debug": fmt.Sprintf("Flushing %v handlers", len(h.handlers))})
				for _, ha := range h.handlers {
					if err := ha.Flush(); err != nil {
						h.logger.Error("heartbeater", err, lager.Data{"handler": ha.Name()})
					}
				}
			case name := <-h.counter:
				for _, ha := range h.handlers {
					ha.Handle(name)
				}
			case <-h.done:
				h.logger.Info("heartbeater", lager.Data{"debug": fmt.Sprintf("Heartbeat polling done for %v handlers", len(h.handlers))})
				for _, ha := range h.handlers {
					h.logger.Info("heartbeater", lager.Data{"debug": "Flushing", "handler": ha.Name()})
					if err := ha.Flush(); err != nil {
						h.logger.Error("heartbeater", err, lager.Data{"handler": ha.Name()})
					}
				}
				h.logger.Info("heartbeater", lager.Data{"debug": "all handlers flushed"})
				return
			}
		}
	}()
}

func (h *heartbeater) Increment(name string) {
	if h.started {
		h.counter <- name
	} else {
		h.logger.Error(
			"heartbeater",
			errors.New("attempted to increment counter without starting heartbeater"),
		)
	}
}

func (h *heartbeater) Stop() {
	h.logger.Info("heartbeater", lager.Data{"debug": "Stopping heartbeater"})
	h.done <- struct{}{}
	h.started = false
}
