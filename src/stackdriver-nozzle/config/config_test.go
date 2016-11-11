package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/config"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Setenv("FIREHOSE_ENDPOINT", "https://api.example.com")
		os.Setenv("FIREHOSE_EVENTS", "LogMessage")
		os.Setenv("FIREHOSE_USERNAME", "admin")
		os.Setenv("FIREHOSE_PASSWORD", "monkey123")
		os.Setenv("FIREHOSE_SKIP_SSL", "true")
		os.Setenv("FIREHOSE_SUBSCRIPTION_ID", "my-subscription-id")
		os.Setenv("FIREHOSE_NEWLINE_TOKEN", "∴")
		os.Setenv("GCP_PROJECT_ID", "test")
	})

	It("returns valid config from environment", func() {
		c, err := config.NewConfig()

		Expect(err).To(BeNil())
		Expect(c.APIEndpoint).To(Equal("https://api.example.com"))
	})

	DescribeTable("required values aren't empty", func(envName string) {
		os.Setenv(envName, "")

		_, err := config.NewConfig()

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring(envName))
	},
		Entry("FIREHOSE_ENDPOINT", "FIREHOSE_ENDPOINT"),
		Entry("FIREHOSE_EVENTS", "FIREHOSE_EVENTS"),
		Entry("FIREHOSE_SUBSCRIPTION_ID", "FIREHOSE_SUBSCRIPTION_ID"),
	)
})
