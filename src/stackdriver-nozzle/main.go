package main

import (
	"os"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/version"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
)

func main() {
	a := newApp()

	defer handleFatalError(a)

	producer := a.newProducer()
	consumer := a.newConsumer()

	errs, fhErrs := consumer.Start(producer)
	defer consumer.Stop()

	go func() {
		for err := range errs {
			a.logger.Error("nozzle", err)
		}
	}()

	fatalErr := <-fhErrs
	if fatalErr != nil {
		a.logger.Fatal("firehose", fatalErr)
	}
}

func handleFatalError(a *app) {
	if !a.c.DebugNozzle {
		return
	}

	if e := recover(); e != nil {
		stack := make([]byte, 1<<16)
		stackSize := runtime.Stack(stack, true)
		stackTrace := string(stack[:stackSize])

		payload := map[string]interface{}{
			"serviceContext": map[string]interface{}{
				"service": version.Name,
				"version": version.Release,
			},
			"message": stackTrace,
		}

		log := &stackdriver.Log{
			Payload:  payload,
			Labels:   map[string]string{},
			Severity: logging.Error,
		}

		logAdapter, err := a.newLogAdapter()
		if err == nil {
			logAdapter.PostLog(log)
			logAdapter.Flush()
		} else {
			a.logger.Error("error getting logAdapter", lager.Data{"err": err})
		}

		// Re-throw the error, we want to ensure it's logged directly to
		// stackdriver but we are not in a recoverable state.
		panic(e)
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

	trigger := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	heartbeater := heartbeat.NewHeartbeater(logger, trigger)

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
		logger:      logger,
		c:           c,
		cfConfig:    cfConfig,
		cfClient:    cfClient,
		labelMaker:  labelMaker,
		heartbeater: heartbeater,
	}
}

type app struct {
	logger      lager.Logger
	c           *config.Config
	cfConfig    *cfclient.Config
	cfClient    *cfclient.Client
	labelMaker  nozzle.LabelMaker
	heartbeater heartbeat.Heartbeater
}

func (a *app) newProducer() firehose.Client {
	fhClient := firehose.NewClient(a.cfConfig, a.cfClient, a.c.SubscriptionID)

	producer, err := filter.New(fhClient, strings.Split(a.c.Events, ","), a.heartbeater)
	if err != nil {
		a.logger.Fatal("filter", err)
	}

	return producer
}

func (a *app) newConsumer() *nozzle.Nozzle {
	return &nozzle.Nozzle{
		LogSink:     a.newLogSink(),
		MetricSink:  a.newMetricSink(),
		Heartbeater: a.heartbeater,
	}
}

func (a *app) newLogSink() nozzle.Sink {
	logAdapter, logErrs := a.newLogAdapter()
	go func() {
		err := <-logErrs
		a.logger.Error("logAdapter", err)
	}()

	return nozzle.NewLogSink(a.labelMaker, logAdapter)
}

func (a *app) newLogAdapter() (stackdriver.LogAdapter, <-chan error) {
	return stackdriver.NewLogAdapter(
		a.c.ProjectID,
		a.c.BatchCount,
		time.Duration(a.c.BatchDuration)*time.Second,
		a.heartbeater,
	)
}

func (a *app) newMetricSink() nozzle.Sink {
	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		a.logger.Fatal("metricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(a.c.ProjectID, metricClient, a.heartbeater)
	if err != nil {
		a.logger.Error("metricAdapter", err)
	}

	metricBuffer, errs := stackdriver.NewMetricsBuffer(a.c.BatchCount, metricAdapter)
	go func() {
		for err = range errs {
			a.logger.Error("metricsBuffer", err)
		}
	}()

	return nozzle.NewMetricSink(a.labelMaker, metricBuffer, nozzle.NewUnitParser())
}
