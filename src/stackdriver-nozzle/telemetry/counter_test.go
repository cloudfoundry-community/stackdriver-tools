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

package telemetry_test

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Counter", func() {
	var (
		subject telemetry.Counter
		logger  *mocks.MockLogger
		handler *mocks.MockTelemetrySink
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		handler = &mocks.MockTelemetrySink{}

		subject = telemetry.NewCounter(logger, time.Duration(100*time.Millisecond), handler)
		subject.Start()
	})

	It("should start at zero", func() {
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{}},
			},
		}))
	})

	It("should count events", func() {
		subject.IncrementBy("foo", 10)

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{"foo": 10}},
			},
		}))

		Expect(handler.RecordCounters).To(HaveLen(1))
		Expect(handler.RecordCounters[0]).To(HaveKeyWithValue("foo", 10))
	})

	It("should reset the counter on triggers", func() {
		subject.IncrementBy("foo", 10)
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{"foo": 10}},
			},
		}))

		Expect(handler.RecordCounters).To(HaveLen(1))
		Expect(handler.RecordCounters[0]).To(HaveKeyWithValue("foo", 10))

		subject.IncrementBy("foo", 5)
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{"foo": 5}},
			},
		}))

		Expect(handler.RecordCounters).To(HaveLen(2))
		Expect(handler.RecordCounters[1]).To(HaveKeyWithValue("foo", 5))
	})

	It("should stop counting", func() {
		subject.IncrementBy("foo", 5)
		subject.Stop()

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{"foo": 5}},
			},
		}))

		// The error is reported
		subject.Increment("foo")
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.ERROR,
			Action: telemetry.Action,
			Err:    telemetry.CounterStoppedErr,
		}))

		// The error is not repeated
		logsCount := len(logger.Logs())
		subject.Increment("foo")
		Expect(logsCount).To(Equal(len(logger.Logs())))
	})

	It("can count multiple events", func() {
		subject.IncrementBy("baz", 15)
		subject.IncrementBy("foo", 10)
		subject.IncrementBy("bar", 5)

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: telemetry.Action,
			Datas: []lager.Data{
				{"counters": map[string]int{
					"foo": 10,
					"bar": 5,
					"baz": 15,
				}},
			},
		}))

		Expect(handler.RecordCounters).To(HaveLen(1))
		counters := handler.RecordCounters[0]
		Expect(counters).To(HaveKeyWithValue("foo", 10))
		Expect(counters).To(HaveKeyWithValue("bar", 5))
		Expect(counters).To(HaveKeyWithValue("baz", 15))
	})

	Context("with a slow handler", func() {
		var (
			handlePosted chan struct{}
		)
		BeforeEach(func() {
			handlePosted = make(chan struct{})
			handler.RecordFn = func(map[string]int) {
				handlePosted <- struct{}{}
				time.Sleep(5 * time.Second)
			}
		})
		It("isn't blocked", func() {
			subject.IncrementBy("foo", 5)
			Eventually(handlePosted).Should(Receive())

			unblocked := make(chan struct{})
			go func() {
				subject.IncrementBy("foo", 2)
				unblocked <- struct{}{}
			}()
			Eventually(unblocked).Should(Receive())
		})
	})
})
