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

package session_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/fakes"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/session"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Session", func() {
	Context("run", func() {
		It("emits logs", func() {
			writer := fakes.Writer{}
			emitter := cloudfoundry.NewEmitter(&writer, 1, 0)
			probe := &fakes.LosslessProbe{}
			s := session.NewSession(emitter, probe)
			_, err := s.Run(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(writer.Writes)).To(Equal(1))
		})

		It("fails if an error occurs while emitting logs", func() {
			err := errors.New("failed to write")
			writer := fakes.FailingWriter{
				Err: err,
			}
			emitter := cloudfoundry.NewEmitter(&writer, 1, 0)
			probe := &fakes.LosslessProbe{}
			s := session.NewSession(emitter, probe)
			_, retErr := s.Run(0)
			Expect(retErr).To(HaveOccurred())
			Expect(retErr).To(Equal(err))
		})

		It("correctly reports zero loss", func() {
			writer := fakes.Writer{}
			emitter := cloudfoundry.NewEmitter(&writer, 10, 0)
			probe := &fakes.LosslessProbe{}
			s := session.NewSession(emitter, probe)
			r, err := s.Run(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Loss).To(Equal(0.0))

		})

		It("correctly reports 50% loss", func() {
			writer := fakes.Writer{}
			emitter := cloudfoundry.NewEmitter(&writer, 10, 0)
			probe := &fakes.ConfigurableProbe{
				FindFunc: func(_ time.Time, _ string, _ int) (int, error) {
					return 5, nil

				}}
			s := session.NewSession(emitter, probe)
			r, err := s.Run(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Loss).To(Equal(0.5))

		})

		It("correctly detects an error in probe.Find", func() {
			writer := fakes.Writer{}
			emitter := cloudfoundry.NewEmitter(&writer, 10, 0)
			err := errors.New("error while trying to find ")
			probe := &fakes.ConfigurableProbe{
				FindFunc: func(_ time.Time, _ string, _ int) (int, error) {
					return 5, err
				}}
			s := session.NewSession(emitter, probe)
			_, returnErr := s.Run(0)
			Expect(returnErr).To(HaveOccurred())
			Expect(returnErr).To(Equal(err))
		})

		It("Creates a different needle on each call", func() {
			writer := fakes.Writer{}
			emitter := cloudfoundry.NewEmitter(&writer, 1, 0)
			probe := &fakes.LosslessProbe{}
			s := session.NewSession(emitter, probe)
			_, err := s.Run(0)
			Expect(err).ToNot(HaveOccurred())
			_, err = s.Run(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Writes[0]).ToNot(Equal(writer.Writes[1]))
		})

	})
})
