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
	"regexp"
	"sync"

	"github.com/cloudfoundry/sonde-go/events"
)

type matcher func(*events.Envelope, *regexp.Regexp) bool

const (
	// MatchName matches the supplied regexp against the Envelope
	// origin and the metric name concatenated with ".", e.g.
	// "gorouter.total_requests".
	MatchName = "name"
	// MatchJob matches the supplied regexp against the Envelope job.
	MatchJob = "job"
)

var matchTypes = map[string]matcher{
	MatchName: matchName,
	MatchJob:  matchJob,
}

// An EventFilter can be used to specify regular expressions to match
// against event proto fields, for the purpose of blacklisting or
// whitelisting nozzle processing of firehose events.
type EventFilter struct {
	matchers []func(*events.Envelope) bool
	mu       sync.RWMutex
}

// Add adds a regular expression to the filter, matching against a particular
// set of event proto fields based on the match type.
func (ef *EventFilter) Add(mt, re string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()
	matchFunc, ok := matchTypes[mt]
	if !ok {
		return fmt.Errorf("unrecognized match type %q", mt)
	}
	compiled, err := regexp.Compile(re)
	if err != nil {
		return err
	}
	ef.matchers = append(ef.matchers, func(event *events.Envelope) bool {
		return matchFunc(event, compiled)
	})
	return nil
}

// Match returns true if the provided event Envelope matches any
// of the filters added to the MetricFilter.
func (ef *EventFilter) Match(event *events.Envelope) bool {
	if ef == nil {
		// Allow nil to be passed as an empty filter.
		return false
	}
	ef.mu.RLock()
	defer ef.mu.RUnlock()
	for _, matchfunc := range ef.matchers {
		if matchfunc(event) {
			return true
		}
	}
	return false
}

// Len returns the number of filters added to the EventFilter.
// Useful for external testing.
func (ef *EventFilter) Len() int {
	if ef == nil {
		return 0
	}
	return len(ef.matchers)
}

func getName(event *events.Envelope) string {
	switch event.GetEventType() {
	case events.Envelope_ValueMetric:
		return event.GetValueMetric().GetName()
	case events.Envelope_CounterEvent:
		return event.GetCounterEvent().GetName()
	}
	// ContainerMetric is absent from the above list because it
	// results in 5 metrics and doesn't really have one "name".
	// ContainerMetrics can be blacklisted as an event type
	// in the filter sink anyway, so this is probably fine.
	return ""
}

func matchName(event *events.Envelope, re *regexp.Regexp) bool {
	origin, name := event.GetOrigin(), getName(event)
	if origin == "" || name == "" {
		return false
	}
	return re.MatchString(fmt.Sprintf("%s.%s", origin, name))
}

func matchJob(event *events.Envelope, re *regexp.Regexp) bool {
	return re.MatchString(event.GetJob())
}
