/*
 * Copyright 2019 Google Inc.
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

package cloudfoundry_test

import (
	"errors"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Emitter", func() {
	It("logs to stdout once", func() {
		mockWriter := fakes.Writer{}

		writer := cloudfoundry.NewEmitter(&mockWriter, 1, 0)
		writer.Emit("something")

		Expect(mockWriter.Writes).To(HaveLen(1))
		Expect(mockWriter.Writes[0]).To(ContainSubstring("something"))
	})

	It("logs to stdout x specified times", func() {
		mockWriter := fakes.Writer{}

		writer := cloudfoundry.NewEmitter(&mockWriter, 10, 0)
		writer.Emit("something")

		Expect(mockWriter.Writes).To(HaveLen(10))
	})

	It("returns a count of successfully emitted logs", func() {
		mockWriter := fakes.Writer{}

		writer := cloudfoundry.NewEmitter(&mockWriter, 10, 0)
		count, _ := writer.Emit("something")

		Expect(count).To(Equal(10))
	})

	It("returns zero when no logs are emitted", func() {
		mockWriter := fakes.FailingWriter{}
		mockWriter.Err = errors.New("Fail!!")

		writer := cloudfoundry.NewEmitter(&mockWriter, 10, 0)
		count, _ := writer.Emit("something")

		Expect(count).To(Equal(0))
	})
})
