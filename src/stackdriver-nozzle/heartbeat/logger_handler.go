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
	"sync"

	"github.com/cloudfoundry/lager"
)

type loggerHandler struct {
	logger lager.Logger
	prefix string

	counterMu sync.Mutex // Guards counter
	counter   map[string]uint
}

func NewLoggerHandler(logger lager.Logger, prefix string) *loggerHandler {
	return &loggerHandler{
		logger:  logger,
		counter: map[string]uint{},
		prefix:  prefix,
	}
}

func (h *loggerHandler) Name() string {
	return "loggerHandler"
}

func (h *loggerHandler) Handle(event string) {
	h.counterMu.Lock()
	defer h.counterMu.Unlock()
	h.counter[event]++
	return
}

func (h *loggerHandler) Flush() error {
	h.counterMu.Lock()
	defer h.counterMu.Unlock()
	h.logger.Info(
		h.prefix, lager.Data{"counters": h.counter},
	)
	h.counter = map[string]uint{}
	return nil
}
