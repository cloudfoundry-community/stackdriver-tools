package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"github.com/cloudfoundry/lager"
)

func main() {
	a := newApp()

	ctx, cancel := context.WithCancel(context.Background())

	if a.c.DebugNozzle {
		defer handleFatalError(a, cancel)

		go func() {
			a.logger.Info("pprof", lager.Data{
				"http.ListenAndServe": http.ListenAndServe("localhost:6060", nil),
			})
		}()
	}

	producer := a.newProducer()
	consumer, err := a.newConsumer(ctx)
	if err != nil {
		a.logger.Fatal("construction", err)
	}

	errs, fhErrs := consumer.Start(producer)
	defer consumer.Stop()

	go func() {
		for err := range errs {
			a.logger.Error("nozzle", err)
		}
	}()

	if fatalErr := <-fhErrs; fatalErr != nil {
		cancel()
		t := time.NewTimer(5 * time.Second)
		for {
			select {
			case <-time.Tick(100 * time.Millisecond):
				if a.bufferEmpty() {
					a.logger.Fatal("firehose", fatalErr, lager.Data{"cleanup": "The metrics buffer was successfully flushed before shutdown"})
				}
			case <-t.C:
				a.logger.Fatal("firehose", fatalErr, lager.Data{"cleanup": "The metrics buffer could not be flushed before shutdown"})
			}
		}
	}
}

func handleFatalError(a *app, cancel context.CancelFunc) {
	if e := recover(); e != nil {
		// Cancel the context
		cancel()

		stack := make([]byte, 1<<16)
		stackSize := runtime.Stack(stack, true)
		stackTrace := string(stack[:stackSize])

		payload := map[string]interface{}{
			"serviceContext": map[string]interface{}{
				"service": version.Name,
				"version": version.Release(),
			},
			"message": fmt.Sprintf("%v\n%v", e, stackTrace),
		}

		log := &stackdriver.Log{
			Payload:  payload,
			Labels:   map[string]string{},
			Severity: logging.Error,
		}

		// Purposefully get a new log adapter here since there
		// were issues re-using the one that the nozzle uses.
		logAdapter, _ := a.newLogAdapter()
		logAdapter.PostLog(log)
		logAdapter.Flush()

		// Re-throw the error, we want to ensure it's logged directly to
		// stackdriver but we are not in a recoverable state.
		panic(e)
	}
}

func newApp() *app {
	logger := lager.NewLogger("stackdriver-nozzle")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.Info("version", lager.Data{"name": version.Name, "release": version.Release(), "user_agent": version.UserAgent()})

	c, err := config.NewConfig()
	if err != nil {
		logger.Fatal("config", err)
	}

	logger.Info("arguments", c.ToData())

	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		logger.Fatal("metricClient", err)
	}

	// Create a metricAdapter that will be used by the heartbeater
	// to send heartbeat metrics to Stackdriver. This metricAdapter
	// has its own heartbeater (with its own trigger) that writes to a logger.
	trigger := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	adapterHeartbeater := heartbeat.NewHeartbeater(logger, trigger)
	adapterHeartbeater.Start()
	metricAdapter, err := stackdriver.NewMetricAdapter(c.ProjectID, metricClient, adapterHeartbeater)
	if err != nil {
		logger.Error("metricAdapter", err)
	}

	// Create a heartbeater that will write heartbeat events to Stackdriver
	// logging and monitoring. It uses the metricAdapter created previously
	// to write to Stackdriver.
	metricHandler := heartbeat.NewMetricHandler(metricAdapter, logger, c.NozzleId, c.NozzleName, c.NozzleZone)
	trigger2 := time.NewTicker(time.Duration(c.HeartbeatRate) * time.Second).C
	heartbeater := heartbeat.NewLoggerMetricHeartbeater(metricHandler, logger, trigger2)

	cfConfig := &cfclient.Config{
		ApiAddress:        c.APIEndpoint,
		Username:          c.Username,
		Password:          c.Password,
		SkipSslValidation: c.SkipSSL}
	cfClient, err := cfclient.NewClient(cfConfig)
	if err != nil {
		logger.Error("cfClient", err)
	}

	var appInfoRepository cloudfoundry.AppInfoRepository
	if c.ResolveAppMetadata {
		appInfoRepository = cloudfoundry.NewAppInfoRepository(cfClient)
	} else {
		appInfoRepository = cloudfoundry.NullAppInfoRepository()
	}
	labelMaker := nozzle.NewLabelMaker(appInfoRepository)

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
	bufferEmpty func() bool
}

func (a *app) newProducer() cloudfoundry.Firehose {
	return cloudfoundry.NewFirehose(a.cfConfig, a.cfClient, a.c.SubscriptionID)
}

func (a *app) newConsumer(ctx context.Context) (*nozzle.Nozzle, error) {
	loggingEvents := strings.Split(a.c.LoggingEvents, ",")
	metricEvents := strings.Split(a.c.MonitoringEvents, ",")

	logSink, err := nozzle.NewFilterSink(loggingEvents, a.newLogSink())
	if err != nil {
		return nil, err
	}

	metricSink, err := nozzle.NewFilterSink(metricEvents, a.newMetricSink(ctx))
	if err != nil {
		return nil, err
	}

	return &nozzle.Nozzle{
		LogSink:     logSink,
		MetricSink:  metricSink,
		Heartbeater: a.heartbeater,
	}, nil
}

func (a *app) newLogSink() nozzle.Sink {
	logAdapter, logErrs := a.newLogAdapter()
	go func() {
		err := <-logErrs
		a.logger.Error("logAdapter", err)
	}()

	return nozzle.NewLogSink(a.labelMaker, logAdapter, a.c.NewlineToken)
}

func (a *app) newLogAdapter() (stackdriver.LogAdapter, <-chan error) {
	return stackdriver.NewLogAdapter(
		a.c.ProjectID,
		a.c.BatchCount,
		time.Duration(a.c.BatchDuration)*time.Second,
		a.heartbeater,
	)
}

func (a *app) newMetricSink(ctx context.Context) nozzle.Sink {
	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		a.logger.Fatal("metricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(a.c.ProjectID, metricClient, a.heartbeater)
	if err != nil {
		a.logger.Error("metricAdapter", err)
	}

	metricBuffer, errs := stackdriver.NewAutoCulledMetricsBuffer(ctx, a.logger, time.Duration(a.c.MetricsBufferDuration)*time.Second, a.c.MetricsBufferSize, metricAdapter)
	a.bufferEmpty = metricBuffer.IsEmpty
	go func() {
		for err = range errs {
			a.logger.Error("metricsBuffer", err)
		}
	}()

	return nozzle.NewMetricSink(a.labelMaker, metricBuffer, nozzle.NewUnitParser())
}
