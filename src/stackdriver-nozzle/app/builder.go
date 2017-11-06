package app

import (
	"context"
	_ "net/http/pprof"
	"strings"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/metrics_pipeline"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"github.com/cloudfoundry/lager"
)

type App struct {
	logger      lager.Logger
	c           *config.Config
	cfConfig    *cfclient.Config
	cfClient    *cfclient.Client
	labelMaker  nozzle.LabelMaker
	heartbeater heartbeat.Heartbeater
	bufferEmpty func() bool
}

func New(c *config.Config, logger lager.Logger) *App {
	logger.Info("version", lager.Data{"name": version.Name, "release": version.Release(), "user_agent": version.UserAgent()})
	logger.Info("arguments", c.ToData())

	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		logger.Fatal("metricClient", err)
	}

	// Create a metricAdapter that will be used by the heartbeater
	// to send heartbeat metrics to Stackdriver. This metricAdapter
	// has its own heartbeater (with its own trigger) that writes to a logger.
	trigger := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	adapterHeartbeater := heartbeat.NewHeartbeater(logger, trigger, "heartbeater.telemetry.emitted")
	adapterHeartbeater.Start()
	metricAdapter, err := stackdriver.NewMetricAdapter(c.ProjectID, metricClient, c.MetricsBatchSize, adapterHeartbeater, logger)
	if err != nil {
		logger.Fatal("metricAdapter", err)
	}

	// Create a heartbeater that will write heartbeat events to Stackdriver
	// logging and monitoring. It uses the metricAdapter created previously
	// to write to Stackdriver.
	metricHandler := heartbeat.NewMetricHandler(metricAdapter, logger, c.NozzleId, c.NozzleName, c.NozzleZone)
	trigger2 := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	heartbeater := heartbeat.NewLoggerMetricHeartbeater(metricHandler, logger, trigger2, "heartbeater.telemetry")

	cfConfig := &cfclient.Config{
		ApiAddress:        c.APIEndpoint,
		Username:          c.Username,
		Password:          c.Password,
		SkipSslValidation: c.SkipSSL}
	cfClient, err := cfclient.NewClient(cfConfig)
	if err != nil {
		logger.Fatal("cfClient", err)
	}

	var appInfoRepository cloudfoundry.AppInfoRepository
	if c.ResolveAppMetadata {
		appInfoRepository = cloudfoundry.NewAppInfoRepository(cfClient)
	} else {
		appInfoRepository = cloudfoundry.NullAppInfoRepository()
	}
	labelMaker := nozzle.NewLabelMaker(appInfoRepository)

	return &App{
		logger:      logger,
		c:           c,
		cfConfig:    cfConfig,
		cfClient:    cfClient,
		labelMaker:  labelMaker,
		heartbeater: heartbeater,
	}
}

func (a *App) newProducer() cloudfoundry.Firehose {
	return cloudfoundry.NewFirehose(a.cfConfig, a.cfClient, a.c.SubscriptionID)
}

func (a *App) newConsumer(ctx context.Context) (*nozzle.Nozzle, error) {
	logEvents, err := nozzle.ParseEvents(strings.Split(a.c.LoggingEvents, ","))
	if err != nil {
		return nil, err
	}

	metricEvents, err := nozzle.ParseEvents(strings.Split(a.c.MonitoringEvents, ","))
	if err != nil {
		return nil, err
	}

	logAdapter := a.newLogAdapter()
	filteredLogSink, err := nozzle.NewFilterSink(logEvents, nozzle.NewLogSink(a.labelMaker, logAdapter, a.c.NewlineToken))
	if err != nil {
		return nil, err
	}

	// Destination for metrics
	metricAdapter := a.newMetricAdapter()
	// Routes metrics to Stackdriver Logging/Stackdriver Monitoring
	metricRouter := metrics_pipeline.NewRouter(metricAdapter, metricEvents, logAdapter, logEvents)
	// Handles and translates Firehose events. Performs buffering/culling.
	metricSink := a.newMetricSink(ctx, metricRouter)
	// Filter Firehose events to what the user selects
	metricRouterEvents := append(logEvents, metricEvents...)
	filteredMetricSink, err := nozzle.NewFilterSink(metricRouterEvents, metricSink)
	if err != nil {
		return nil, err
	}

	return &nozzle.Nozzle{
		LogSink:     filteredLogSink,
		MetricSink:  filteredMetricSink,
		Heartbeater: a.heartbeater,
	}, nil
}

func (a *App) newLogAdapter() stackdriver.LogAdapter {
	logAdapter, logErrs := stackdriver.NewLogAdapter(
		a.c.ProjectID,
		a.c.LoggingBatchCount,
		time.Duration(a.c.LoggingBatchDuration)*time.Second,
		a.heartbeater,
	)
	go func() {
		err := <-logErrs
		a.logger.Error("logAdapter", err)
	}()

	return logAdapter
}

func (a *App) newMetricAdapter() stackdriver.MetricAdapter {
	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		a.logger.Fatal("metricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(a.c.ProjectID, metricClient, a.c.MetricsBatchSize, a.heartbeater, a.logger)
	if err != nil {
		a.logger.Fatal("metricAdapter", err)
	}

	return metricAdapter
}

func (a *App) newMetricSink(ctx context.Context, metricAdapter stackdriver.MetricAdapter) nozzle.Sink {
	metricBuffer := metrics_pipeline.NewAutoCulledMetricsBuffer(ctx, a.logger, time.Duration(a.c.MetricsBufferDuration)*time.Second, metricAdapter, a.heartbeater)
	a.bufferEmpty = metricBuffer.IsEmpty

	return nozzle.NewMetricSink(a.labelMaker, metricBuffer, nozzle.NewUnitParser())
}
