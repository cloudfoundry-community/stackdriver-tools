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

package heartbeat_test

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeater", func() {
	var (
		subject heartbeat.Heartbeater
		logger  *mocks.MockLogger
		handler *mocks.MockHandler
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		handler = &mocks.MockHandler{}

		subject = heartbeat.NewTelemetry(logger, time.Duration(100*time.Millisecond), handler)
		subject.Start()
	})

	It("should start at zero", func() {
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{}},
			},
		}))
	})

	It("should count events", func() {
		subject.IncrementBy("foo", 10)

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 10}},
			},
		}))

		Expect(handler.HandleCount).To(Equal(1))
		Expect(handler.FlushCount).To(Equal(1))
	})

	It("should reset the heartbeater on triggers", func() {
		subject.IncrementBy("foo", 10)
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 10}},
			},
		}))

		subject.IncrementBy("foo", 5)
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 5}},
			},
		}))

		Expect(handler.HandleCount).To(Equal(2))
		Expect(handler.FlushCount).To(Equal(2))
	})

	It("should stop counting", func() {
		subject.IncrementBy("foo", 5)
		subject.Stop()

		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 5}},
			},
		}))

		// The error is reported
		subject.Increment("foo")
		Eventually(logger.Logs).Should(ContainElement(mocks.Log{
			Level:  lager.ERROR,
			Action: "heartbeater",
			Err:    heartbeat.HeartbeaterStoppedErr,
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
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{
					"foo": 10,
					"bar": 5,
					"baz": 15,
				}},
			},
		}))

		Expect(handler.FlushCount).To(Equal(1))
		Expect(handler.HandleCount).To(Equal(3))
	})

	Context("with a slow handler", func() {
		var (
			handlePosted chan struct{}
		)
		BeforeEach(func() {
			handlePosted = make(chan struct{})
			handler.HandleFn = func(string, uint) {
				handlePosted <- struct{}{}
				time.Sleep(5 * time.Second)
			}
			handler.FlushFn = func() {
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
