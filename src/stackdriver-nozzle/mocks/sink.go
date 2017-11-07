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

package mocks

import (
	"sync"

	"github.com/cloudfoundry/sonde-go/events"
)

type Sink struct {
	HandledEnvelopes []events.Envelope
	mutex            sync.Mutex
}

func (s *Sink) Receive(envelope *events.Envelope) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.HandledEnvelopes = append(s.HandledEnvelopes, *envelope)
}

func (s *Sink) LastEnvelope() *events.Envelope {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.HandledEnvelopes) == 0 {
		return nil
	}

	return &s.HandledEnvelopes[len(s.HandledEnvelopes)-1]
}
