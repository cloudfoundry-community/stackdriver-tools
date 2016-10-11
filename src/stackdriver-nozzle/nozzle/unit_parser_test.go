package nozzle_test

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UnitParser", func() {
	var AssertUnitParsed func(string, string)

	BeforeEach(func() {
		AssertUnitParsed = func(input, output string) {
			Expect(nozzle.UnitParser(input)).To(Equal(output))
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
			input string
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
			input string
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
			input string
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
})
