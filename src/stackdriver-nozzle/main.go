package main

import (
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/go-cfclient"

	"stackdriver-nozzle/filter"
	"stackdriver-nozzle/firehose"
	"stackdriver-nozzle/nozzle"
	"stackdriver-nozzle/serializer"
	"stackdriver-nozzle/stackdriver"

	"fmt"
	"os"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
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
	boltDatabasePath = kingpin.Flag("boltdb-path", "bolt Database path").
				Default("cached-app-metadata.db").
				OverrideDefaultFromEnvar("BOLTDB_PATH").
				String()
	resolveCfMetadata = kingpin.Flag("resolve-cf-metadata", "resolve CloudFoundry app metadata (eg appName) in log output").
				Default("true").
				OverrideDefaultFromEnvar("RESOLVE_CF_METADATA").
				Bool()
)

func main() {
	kingpin.Parse()

	cfConfig := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		Username:          *username,
		Password:          *password,
		SkipSslValidation: *skipSSLValidation}
	cfClient := cfclient.NewClient(cfConfig)
	input := firehose.NewClient(cfConfig, cfClient)
	sdClient := stackdriver.NewClient(*projectID, *batchCount, *batchDuration)

	var cachingClient caching.Caching
	if *resolveCfMetadata {
		cachingClient = caching.NewCachingBolt(cfClient, *boltDatabasePath)
		// Initialize the caching client with the state of the world
		cachingClient.GetAllApp()
	} else {
		fmt.Println("Not resolving CloudFoundry app metadata")
	}

	output := nozzle.Nozzle{
		StackdriverClient: sdClient,
		Serializer:        serializer.NewSerializer(cachingClient),
	}

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
