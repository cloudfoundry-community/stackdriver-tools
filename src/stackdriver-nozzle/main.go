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
	a := newApp()

	producer := a.producer()
	consumer := a.consumer()

	errs, fhErrs := consumer.Start(producer)
	go func() {
		for err := range errs {
			a.logger.Error("nozzle", err)
		}
	}()

	fatalErr := <-fhErrs
	if fatalErr != nil {
		consumer.Stop()
		a.logger.Fatal("firehose", fatalErr)
	}
}

func newApp() *app {
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

	var cachingClient caching.Caching
	if c.ResolveAppMetadata {
		cachingClient = caching.NewCachingBolt(cfClient, c.BoltDBPath)
	} else {
		cachingClient = caching.NewCachingEmpty()
	}
	cachingClient.CreateBucket()
	labelMaker := nozzle.NewLabelMaker(cachingClient)

	return &app{
		logger:     logger,
		c:          c,
		cfConfig:   cfConfig,
		cfClient:   cfClient,
		labelMaker: labelMaker,
	}
}

type app struct {
	logger     lager.Logger
	c          *config.Config
	cfConfig   *cfclient.Config
	cfClient   *cfclient.Client
	labelMaker nozzle.LabelMaker
}

func (a *app) producer() firehose.Client {
	fhClient := firehose.NewClient(a.cfConfig, a.cfClient, a.c.SubscriptionID)

	producer, err := filter.New(fhClient, strings.Split(a.c.Events, ","))
	if err != nil {
		a.logger.Fatal("filter", err)
	}

	return producer
}

func (a *app) consumer() *nozzle.Nozzle {
	trigger := time.NewTicker(time.Duration(a.c.HeartbeatRate) * time.Second).C
	heartbeater := heartbeat.NewHeartbeat(a.logger, trigger)

	return &nozzle.Nozzle{
		LogSink:     a.logSink(),
		MetricSink:  a.metricSink(),
		Heartbeater: heartbeater,
	}
}

func (a *app) logSink() nozzle.Sink {
	logAdapter, logErrs := stackdriver.NewLogAdapter(
		a.c.ProjectID,
		a.c.BatchCount,
		time.Duration(a.c.BatchDuration)*time.Second,
	)
	go func() {
		err := <-logErrs
		a.logger.Fatal("logAdapter", err)
	}()

	return nozzle.NewLogSink(a.labelMaker, logAdapter)
}

func (a *app) metricSink() nozzle.Sink {
	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		a.logger.Fatal("metricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(a.c.ProjectID, metricClient)
	if err != nil {
		a.logger.Fatal("metricAdapter", err)
	}

	metricBuffer, errs := stackdriver.NewMetricsBuffer(a.c.BatchCount, metricAdapter)
	go func() {
		for err = range errs {
			a.logger.Error("metricsBuffer", err)
		}
	}()

	return nozzle.NewMetricSink(a.labelMaker, metricBuffer, nozzle.NewUnitParser())
}
