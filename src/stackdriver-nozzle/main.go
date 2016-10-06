package main

import (
	"os"
	"strings"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"

	"stackdriver-nozzle/filter"
	"stackdriver-nozzle/firehose"
	"stackdriver-nozzle/nozzle"
	"stackdriver-nozzle/serializer"
	"stackdriver-nozzle/stackdriver"

	"github.com/cloudfoundry-community/go-cfclient"
	"gopkg.in/alecthomas/kingpin.v2"

	"time"

	"github.com/cloudfoundry/lager"
	"stackdriver-nozzle/heartbeat"
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
	const triggerDuration = 30 * time.Second
	kingpin.Parse()

	logger := lager.NewLogger("stackdriver-nozzle")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.Info("arguments", lager.Data{
		"resolveCfMetadata": resolveCfMetadata,
		"events":            eventsFilter,
	})

	cfConfig := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		Username:          *username,
		Password:          *password,
		SkipSslValidation: *skipSSLValidation}
	cfClient := cfclient.NewClient(cfConfig)
	input := firehose.NewClient(cfConfig, cfClient, logger)

	var cachingClient caching.Caching
	if *resolveCfMetadata {
		cachingClient = caching.NewCachingBolt(cfClient, *boltDatabasePath)
	} else {
		cachingClient = caching.NewCachingEmpty()
	}

	trigger := time.NewTicker(triggerDuration).C
	heartbeater := heartbeat.NewHeartbeat(logger, trigger)
	sdClient := stackdriver.NewClient(*projectID, *batchCount, *batchDuration, logger, heartbeater)
	nozzleSerializer := serializer.NewSerializer(cachingClient, logger)

	output := nozzle.Nozzle{
		StackdriverClient: sdClient,
		Serializer:        nozzleSerializer,
	}

	filteredOutput, err := filter.New(&output, strings.Split(*eventsFilter, ","))
	if err != nil {
		logger.Fatal("newFilter", err)
	}

	heartbeater.Start()
	err = input.StartListening(filteredOutput)
	heartbeater.Stop()

	if err != nil {
		logger.Fatal("firehoseStart", err)
	}
}
