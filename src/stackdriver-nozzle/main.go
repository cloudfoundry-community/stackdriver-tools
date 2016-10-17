package main

import (
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
)

func main() {
	logger := lager.NewLogger("stackdriver-nozzle")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	c, err := config.NewConfig()
	if err != nil {
		logger.Fatal("config", err)
	}

	logger.Info("arguments", c.ToData())

	cfConfig := &cfclient.Config{
		ApiAddress:        c.APIEndpoint,
		Username:          c.Username,
		Password:          c.Password,
		SkipSslValidation: c.SkipSSL}
	cfClient := cfclient.NewClient(cfConfig)
	fhClient := firehose.NewClient(cfConfig, cfClient, c.SubscriptionID)

	var cachingClient caching.Caching
	if c.ResolveAppMetadata {
		cachingClient = caching.NewCachingBolt(cfClient, c.BoltDBPath)
	} else {
		cachingClient = caching.NewCachingEmpty()
	}
	cachingClient.CreateBucket()

	logAdapter, logErrs := stackdriver.NewLogAdapter(
		c.ProjectID,
		c.BatchCount,
		time.Duration(c.BatchDuration)*time.Second,
	)
	go func() {
		err := <-logErrs
		logger.Fatal("logAdapter", err)
	}()

	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		logger.Fatal("metricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(c.ProjectID, metricClient)
	if err != nil {
		logger.Fatal("metricAdapter", err)
	}

	metricBuffer, errs := stackdriver.NewMetricsBuffer(c.BatchCount, metricAdapter)
	go func() {
		for err = range errs {
			logger.Error("metricsBuffer", err)
		}
	}()

	trigger := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	heartbeater := heartbeat.NewHeartbeat(logger, trigger)
	labelMaker := nozzle.NewLabelMaker(cachingClient)
	logHandler := nozzle.NewLogSink(labelMaker, logAdapter)
	metricHandler := nozzle.NewMetricSink(labelMaker, metricBuffer, nozzle.NewUnitParser())

	filteredFirehose, err := filter.New(fhClient, strings.Split(c.Events, ","))
	if err != nil {
		logger.Fatal("filter", err)
	}

	nozz := nozzle.Nozzle{
		Firehose:    filteredFirehose,
		LogSink:     logHandler,
		MetricSink:  metricHandler,
		Heartbeater: heartbeater,
	}

	errs, fhErrs := nozz.Start()
	go func() {
		for err := range errs {
			logger.Error("nozzle", err)
		}
	}()

	fatalErr := <-fhErrs
	if fatalErr != nil {
		nozz.Stop()
		logger.Fatal("firehose", fatalErr)
	}
}
