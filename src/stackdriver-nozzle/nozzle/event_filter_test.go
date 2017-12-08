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
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventFilter", func() {
	var (
		subject *EventFilter
		value   float64 = 0.0
		counter uint64  = 0
		unit            = "ms"
	)

	BeforeEach(func() {
		subject = &EventFilter{}
	})

	It("matches names", func() {
		Expect(subject.Add(MatchName, `[^.]+\.total_requests`)).To(BeNil())
		Expect(subject.Add(MatchName, `gorouter\..*`)).To(BeNil())
		Expect(subject.matchers).To(HaveLen(2))

		tests := []struct {
			origin, name string
			match        bool
		}{
			{"", "", false},
			{"", "total_requests", false}, // No origin.
			{"gorouter", "", false},       // No name.
			{"foo", "bar", false},
			{"foo", "total_requests", true},
			{"gorouter", "bar", true},
			{"gorouter", "total_requests", true},
		}

		for _, t := range tests {
			vm := &events.Envelope{
				Origin:    &t.origin,
				EventType: events.Envelope_ValueMetric.Enum(),
				ValueMetric: &events.ValueMetric{
					Name:  &t.name,
					Value: &value,
					Unit:  &unit,
				},
			}
			ce := &events.Envelope{
				Origin:    &t.origin,
				EventType: events.Envelope_CounterEvent.Enum(),
				CounterEvent: &events.CounterEvent{
					Name:  &t.name,
					Delta: &counter,
					Total: &counter,
				},
			}

			Expect(subject.Match(vm)).To(Equal(t.match))
			Expect(subject.Match(ce)).To(Equal(t.match))
		}
	})

	It("matches jobs", func() {
		Expect(subject.Add(MatchJob, `etc[dD](_server)?`)).To(BeNil())
		Expect(subject.Add(MatchJob, `^router$`)).To(BeNil())
		Expect(subject.matchers).To(HaveLen(2))

		origin := "gorouter"
		tests := []struct {
			job   string
			match bool
		}{
			{"", false},
			{"foo", false},
			{"gorouter", false}, // Anchors
			{"router", true},
			{"etc", false},
			{"etcd", true},
			{"etcD_server", true},
		}

		for _, t := range tests {
			event := &events.Envelope{
				Origin:    &origin,
				EventType: events.Envelope_CounterEvent.Enum(),
				Job:       &t.job,
			}
			Expect(subject.Match(event)).To(Equal(t.match))
		}
	})

	It("returns an error when the match type is unknown", func() {
		Expect(subject.Add("foo", `bar`)).NotTo(BeNil())
	})

	It("returns an error when the regexp doesn't compile", func() {
		Expect(subject.Add(MatchJob, `$))]([{{{{(((^^^`)).NotTo(BeNil())
	})
})
