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

package app

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Builder", func() {
	var (
		subject *App
	)

	BeforeEach(func() {
		subject = &App{c: &config.Config{EventFilterJSON: &config.EventFilterJSON{}}}
	})

	DescribeTable("EventFilterJSON to EventFilter",
		func(bl, wl []config.EventFilterRule, lblLen, lwlLen, mblLen, mwlLen int) {
			subject.c.EventFilterJSON.Blacklist = bl
			subject.c.EventFilterJSON.Whitelist = wl

			lbl, lwl, mbl, mwl, err := subject.buildEventFilters()

			Expect(err).To(BeNil())
			Expect(lbl.Len()).To(Equal(lblLen))
			Expect(lwl.Len()).To(Equal(lwlLen))
			Expect(mbl.Len()).To(Equal(mblLen))
			Expect(mwl.Len()).To(Equal(mwlLen))
		},
		Entry("translates nil lists", nil, nil, 0, 0, 0, 0),
		Entry("translates logging blacklist",
			[]config.EventFilterRule{{Type: "name", Sink: "logging", Regexp: ".*"}},
			nil, 1, 0, 0, 0),
		Entry("translates logging whitelist", nil,
			[]config.EventFilterRule{{Type: "name", Sink: "logging", Regexp: ".*"}},
			0, 1, 0, 0),
		Entry("translates monitoring blacklist",
			[]config.EventFilterRule{{Type: "name", Sink: "monitoring", Regexp: ".*"}},
			nil, 0, 0, 1, 0),
		Entry("translates monitoring whitelist", nil,
			[]config.EventFilterRule{{Type: "name", Sink: "monitoring", Regexp: ".*"}},
			0, 0, 0, 1),
		Entry("translates all blacklist",
			[]config.EventFilterRule{{Type: "name", Sink: "all", Regexp: ".*"}},
			nil, 1, 0, 1, 0),
		Entry("translates all whitelist", nil,
			[]config.EventFilterRule{{Type: "name", Sink: "all", Regexp: ".*"}},
			0, 1, 0, 1),
	)

	DescribeTable("chokes on bad EventFilterJSON params",
		func(bl []config.EventFilterRule) {
			subject.c.EventFilterJSON.Blacklist = bl

			lbl, lwl, mbl, mwl, err := subject.buildEventFilters()

			Expect(err).NotTo(BeNil())
			Expect(lbl).To(BeNil())
			Expect(lwl).To(BeNil())
			Expect(mbl).To(BeNil())
			Expect(mwl).To(BeNil())
		},
		Entry("errors on missing sinks", []config.EventFilterRule{{Type: "name", Sink: "", Regexp: ".*"}}),
		Entry("errors on invalid sinks", []config.EventFilterRule{{Type: "name", Sink: "foo", Regexp: ".*"}}),
		Entry("errors on missing types", []config.EventFilterRule{{Type: "", Sink: "all", Regexp: ".*"}}),
		Entry("errors on invalid types", []config.EventFilterRule{{Type: "foo", Sink: "all", Regexp: ".*"}}),
		Entry("errors on missing regexps", []config.EventFilterRule{{Type: "name", Sink: "logging", Regexp: ""}}),
		Entry("errors on invalid regexps", []config.EventFilterRule{{Type: "name", Sink: "logging", Regexp: "$[}}})({"}}),
	)
})
