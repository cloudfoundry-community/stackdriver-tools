package main

import (
	"os"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
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

	client := firehose.NewClient(cfConfig, cfClient, "firehose-rate-script")

	messages, _ := client.Connect()

	trigger := time.Tick(1 * time.Second)
	logger := lager.NewLogger("firehose-rate-script")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	heartbeater := heartbeat.NewHeartbeater(logger, trigger)
	heartbeater.Start()
	defer heartbeater.Stop()
	for _ = range messages {
		heartbeater.Increment("count")
	}
}
