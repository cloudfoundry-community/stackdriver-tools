package main

import (
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
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
	cachingClient.CreateBucket()

	logAdapter, err := stackdriver.NewLogAdapter(*projectID, *batchCount, *batchDuration, logger)
	if err != nil {
		logger.Fatal("newLogAdapter", err)
	}

	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		logger.Fatal("newMetricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(*projectID, metricClient)
	if err != nil {
		logger.Fatal("newMetricAdapter", err)
	}

	trigger := time.NewTicker(triggerDuration).C
	heartbeater := heartbeat.NewHeartbeat(logger, trigger)
	labelMaker := nozzle.NewLabelMaker(cachingClient)
	logHandler := nozzle.NewLogSink(labelMaker, logAdapter)
	metricHandler := nozzle.NewMetricSink(labelMaker, metricAdapter, nozzle.NewUnitParser())

	output := nozzle.Nozzle{
		LogHandler:    logHandler,
		MetricHandler: metricHandler,
		Heartbeater:   heartbeater,
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
