package main

import (
	"os"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
)

func main() {
	apiEndpoint := os.Getenv("FIREHOSE_ENDPOINT")
	username := os.Getenv("FIREHOSE_USERNAME")
	password := os.Getenv("FIREHOSE_PASSWORD")
	_, skipSSLValidation := os.LookupEnv("FIREHOSE_SKIP_SSL")

	cfConfig := &cfclient.Config{
		ApiAddress:        apiEndpoint,
		Username:          username,
		Password:          password,
		SkipSslValidation: skipSSLValidation}

	cfClient := cfclient.NewClient(cfConfig)

	client := cloudfoundry.NewFirehose(cfConfig, cfClient, nil)

	err := client.StartListening(&StdOut{})
	if err != nil {
		panic(err)
	}
}

type StdOut struct{}

func (so *StdOut) HandleEvent(envelope *events.Envelope) error {
	println(envelope.String())
	return nil
}
