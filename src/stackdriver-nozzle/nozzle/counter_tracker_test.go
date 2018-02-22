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
	"context"
	"math"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testCounterTracker(subject *CounterTracker, name string, baseTime time.Time, incoming []uint64, expected []int64) {
	for idx, value := range incoming {
		ts := baseTime.Add(time.Duration(idx) * time.Millisecond)
		total, st := subject.Update(name, value, ts)
		if idx == 0 {
			// First seen value initializes the counter.
			Expect(total).To(BeNumerically("==", 0))
		} else {
			Expect(total).To(BeNumerically("==", expected[idx-1]))
			Expect(st).To(BeTemporally("~", baseTime))
		}
	}
}

var _ = Describe("CounterTracker", func() {
	var (
		subject    *CounterTracker
		counterTTL time.Duration
		logger     *mocks.MockLogger
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		counterTTL = time.Duration(50) * time.Millisecond
		countersExpiredCount.Set(0)
	})

	It("increments counters and handles counter resets", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		subject = NewCounterTracker(ctx, counterTTL, logger)

		incomingTotals := []uint64{10, 15, 25, 40, 10, 20}
		expectedTotals := []int64{5, 15, 30, 40, 50}
		testCounterTracker(subject, "metric", time.Now(), incomingTotals, expectedTotals)
	})

	It("expires old counters", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		subject = NewCounterTracker(ctx, counterTTL, logger)

		incomingTotals := []uint64{150, 165, 165, 170, 200, 200}
		expectedTotals := []int64{15, 15, 20, 50, 50}

		testCounterTracker(subject, "metric2", time.Now(), incomingTotals, expectedTotals)
		Eventually(countersExpiredCount.IntValue).Should(Equal(1))
		testCounterTracker(subject, "metric2", time.Now(), incomingTotals, expectedTotals)
	})

	It("handles int64 overflows", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		subject = NewCounterTracker(ctx, counterTTL, logger)

		baseTime := time.Now()
		incoming := []uint64{150, 165, 165, math.MaxInt64, math.MaxInt64 + 400, math.MaxInt64 + 450}
		expected := []int64{15, 15, 15 + math.MaxInt64 - 165, 400, 450}

		for idx, value := range incoming {
			ts := baseTime.Add(time.Duration(idx) * time.Millisecond)
			total, st := subject.Update("metric3", value, ts)
			if idx == 0 {
				// First seen value initializes the counter.
				Expect(total).To(BeNumerically("==", 0))
				continue
			} else {
				Expect(total).To(BeNumerically("==", expected[idx-1]), "iteration %d", idx)
			}
			// Value at iteration 4 is more than MaxInt64, so start time gets reset.
			if idx < 4 {
				Expect(st).To(BeTemporally("~", baseTime), "iteration %d", idx)
			} else {
				Expect(st).To(BeTemporally("~", baseTime.Add(4*time.Millisecond)), "iteration %d", idx)
			}
		}
	})
})
