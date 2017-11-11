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
	"expvar"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

var (
	intCount *expvar.Int
)

func init() {
	intCount = expvar.NewInt("nozzle.int")
}

var _ = Describe("Reporter", func() {
	var (
		sink     *mocks.TelemetrySink
		reporter telemetry.Reporter
		ctx      context.Context
		cancel   context.CancelFunc
	)

	BeforeEach(func() {
		sink = &mocks.TelemetrySink{}
		reporter = telemetry.NewReporter(1000, sink)
	})

	Context("starting with a single metric", func() {
		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			reporter.Start(ctx)
		})
		AfterEach(func() {
			cancel()
		})

		It("initializes the sink on start", func() {
			Expect(sink.GetInit()).NotTo(BeNil())
			init := sink.GetInit()
			Expect(init).To(HaveLen(1))
			initCountKeyVal := init[0]
			Expect(initCountKeyVal.Key).To(Equal("nozzle.int"))
		})

		It("reports updates", func() {
			intCount.Set(100)
			Eventually(sink.GetLastReport).Should(Not(BeNil()))
			report := sink.GetLastReport()
			Expect(report).To(HaveLen(1))
			initCountKeyVal := report[0]
			Expect(initCountKeyVal.Value.(*expvar.Int).Value()).To(Equal(int64(100)))
		})
	})
})
