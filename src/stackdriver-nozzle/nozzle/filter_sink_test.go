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
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SinkFilter", func() {
	var (
		allEventTypes []events.Envelope_EventType
		sink          *mocks.NozzleSink
	)

	BeforeEach(func() {
		allEventTypes = []events.Envelope_EventType{
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		}
		sink = &mocks.NozzleSink{}
		blacklistedEvents.Set(0)
	})
	It("can accept an empty filter and blocks all events", func() {
		f, err := NewFilterSink([]events.Envelope_EventType{}, nil, nil, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		for _, eventType := range allEventTypes {
			f.Receive(&events.Envelope{EventType: &eventType})
		}

		Expect(sink.HandledEnvelopes).To(BeEmpty())
	})

	It("can accept a single event", func() {
		f, err := NewFilterSink([]events.Envelope_EventType{events.Envelope_LogMessage}, nil, nil, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		eventType := events.Envelope_LogMessage
		event := events.Envelope{EventType: &eventType}

		f.Receive(&event)
		Expect(sink.HandledEnvelopes).To(ContainElement(event))

	})

	It("can accept multiple events to filter", func() {
		f, err := NewFilterSink([]events.Envelope_EventType{events.Envelope_ValueMetric, events.Envelope_LogMessage}, nil, nil, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		for _, eventType := range allEventTypes {
			f.Receive(&events.Envelope{EventType: &eventType})
		}

		Expect(sink.HandledEnvelopes).To(HaveLen(2))
	})

	It("requires a sink", func() {
		f, err := NewFilterSink([]events.Envelope_EventType{}, nil, nil, nil)
		Expect(err).NotTo(BeNil())
		Expect(f).To(BeNil())
	})

	It("blocks blacklisted events", func() {
		bl := &EventFilter{}
		Expect(bl.Add(MatchJob, `router`)).To(BeNil())

		f, err := NewFilterSink([]events.Envelope_EventType{events.Envelope_ValueMetric}, bl, nil, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		routerEvent := events.Envelope{
			EventType: events.Envelope_ValueMetric.Enum(),
			Job:       proto.String("router"),
		}
		f.Receive(&routerEvent)
		for _, job := range []string{"foo", "bar", "baz"} {
			f.Receive(&events.Envelope{
				EventType: events.Envelope_ValueMetric.Enum(),
				Job:       &job,
			})
		}
		f.Receive(&routerEvent)

		Expect(blacklistedEvents.IntValue()).To(Equal(2))
		Expect(sink.HandledEnvelopes).To(HaveLen(3))
		Expect(sink.HandledEnvelopes).NotTo(ContainElement(routerEvent))
	})

	It("doesn't block blacklisted events that are also whitelisted", func() {
		bl, wl := &EventFilter{}, &EventFilter{}
		Expect(bl.Add(MatchJob, `^router$`)).To(BeNil())
		Expect(wl.Add(MatchName, `^gorouter.MetronAgent$`)).To(BeNil())

		f, err := NewFilterSink([]events.Envelope_EventType{events.Envelope_ValueMetric}, bl, wl, sink)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())

		metronEvent := events.Envelope{
			EventType: events.Envelope_ValueMetric.Enum(),
			Job:       proto.String("router"),
			Origin:    proto.String("gorouter"),
			ValueMetric: &events.ValueMetric{
				Name: proto.String("MetronAgent"),
			},
		}
		f.Receive(&metronEvent)
		for _, name := range []string{"foo", "bar", "baz"} {
			f.Receive(&events.Envelope{
				EventType: events.Envelope_ValueMetric.Enum(),
				Job:       proto.String("router"),
				Origin:    proto.String("gorouter"),
				ValueMetric: &events.ValueMetric{
					Name: &name,
				},
			})
		}
		f.Receive(&metronEvent)

		Expect(blacklistedEvents.IntValue()).To(Equal(3))
		Expect(sink.HandledEnvelopes).To(HaveLen(2))
		Expect(sink.HandledEnvelopes).To(ContainElement(metronEvent))
	})

})
