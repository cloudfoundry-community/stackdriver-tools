package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
	"stackdriver-nozzle/filter"
	"stackdriver-nozzle/firehose"
	"stackdriver-nozzle/nozzle"
	"stackdriver-nozzle/stackdriver"
)

var (
	apiEndpoint = kingpin.Flag("api-endpoint",
		"CF API endpoint (use https://api.bosh-lite.com for BOSH Lite)").
		OverrideDefaultFromEnvar("API_ENDPOINT").
		Required().
		String()
	username = kingpin.Flag("username", "username").
			Default("admin").
			OverrideDefaultFromEnvar("FIREHOSE_USERNAME").
			String()
	password = kingpin.Flag("password", "password").
			Default("admin").
			OverrideDefaultFromEnvar("FIREHOSE_PASSWORD").
			String()
	eventsFilter = kingpin.Flag("events", "events to subscribe to from firehose (comma separated)").
			Default("LogMessage,Error").
			OverrideDefaultFromEnvar("FIREHOSE_EVENTS").
			String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "please don't").
				Default("false").
				OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").
				Bool()
	projectID = kingpin.Flag("project-id", "gcp project id").
			OverrideDefaultFromEnvar("PROJECT_ID").
			String() //maybe we can get this from gcp env...? research
	batchCount = kingpin.Flag("batch-count", "maximum number of entries to buffer").
			Default(stackdriver.DefaultBatchCount).
			OverrideDefaultFromEnvar("BATCH_COUNT").
			Int()
	batchDuration = kingpin.Flag("batch-duration", "maximum amount of seconds to buffer").
			Default(stackdriver.DefaultBatchDuration).
			OverrideDefaultFromEnvar("BATCH_DURATION").
			Duration()
)

func main() {
	kingpin.Parse()

	input := firehose.NewClient(*apiEndpoint, *username, *password, *skipSSLValidation)

	sdClient := stackdriver.NewClient(*projectID, *batchCount, *batchDuration)
	output := nozzle.Nozzle{StackdriverClient: sdClient}

	filteredOutput, err := filter.New(&output, strings.Split(*eventsFilter, ","))
	if err != nil {
		if unknownEvent, ok := err.(*filter.UnknownEventName); ok {
			fmt.Printf("Error: %s, possible choices: %s\n", unknownEvent.Error(), strings.Join(unknownEvent.Choices, ","))
			os.Exit(-1)
		} else {
			panic(err)
		}
	}

	fmt.Println("Listening to event(s):", *eventsFilter)

	err = input.StartListening(filteredOutput)

	if err != nil {
		panic(err)
	}
}
