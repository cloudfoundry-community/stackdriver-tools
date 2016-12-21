package nozzle_test

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UnitParser", func() {
	var (
		subject nozzle.UnitParser

		AssertUnitParsed func(string, string)
	)

	BeforeEach(func() {
		subject = nozzle.NewUnitParser()

		AssertUnitParsed = func(input, output string) {
			Expect(subject.Parse(input)).To(Equal(output))
		}
	})

	It("passes through units that don't require translation", func() {
		testCases := []string{
			"s",
			"h",
			"d",
			"ks",
			"Ms",
			"Gs",
			"Ts",
			"Ps",
			"Es",
			"Zs",
			"Ys",
			"ms",
			"ns",
			"ps",
			"fs",
			"as",
			"zs",
			"ys",
			"Kis",
			"Mis",
			"Gis",
			"Tis",
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase, testCase)
		}
	})

	It("translates units that do require translation", func() {
		testCases := []struct {
			input  string
			output string
		}{
			{"b", "bit"},
			{"B", "By"},
			{"M", "min"},
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase.input, testCase.output)
		}
	})

	It("translates invalid units into annotations", func() {
		testCases := []struct {
			input  string
			output string
		}{
			{"count", "{count}"},
			{"invalid word", "{invalid word}"},
			{"foo{bar}baz", "{foobarbaz}"},
			{"oneb", "{oneb}"},
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase.input, testCase.output)
		}
	})

	It("translates units with prefixes", func() {
		testCases := []struct {
			input  string
			output string
		}{
			{"mb", "mbit"},
			{"μs", "us"},
			{"μB", "uBy"},
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase.input, testCase.output)
		}
	})

	It("translates units with expressions", func() {
		testCases := []struct {
			input  string
			output string
		}{
			{"mb/s", "mbit/s"},
			{"μB/M", "uBy/min"},
			{"μB/h", "uBy/h"},
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase.input, testCase.output)
		}
	})

	It("translates units with annotations in expressions", func() {
		testCases := []struct {
			input  string
			output string
		}{
			{"req/s", "{req}/s"},
			{"req/M", "{req}/min"},
			{"mb/joule", "mbit/{joule}"},
		}

		for _, testCase := range testCases {
			AssertUnitParsed(testCase.input, testCase.output)
		}
	})
})
