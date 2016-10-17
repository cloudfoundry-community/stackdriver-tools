package main

import (
	"os"
	"time"

	"math/rand"
	"strings"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
)

var eventTypes = []events.Envelope_EventType{
	events.Envelope_HttpStartStop,
	events.Envelope_LogMessage,
	//events.Envelope_ValueMetric,
	//events.Envelope_CounterEvent,
	events.Envelope_Error,
	//events.Envelope_ContainerMetric,
}

func main() {
	a := newApp()

	logger := lager.NewLogger("load-test")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	trigger := time.NewTicker(time.Second).C
	heartbeater := heartbeat.NewHeartbeat(logger, trigger)

	p := &producer{
		messages:    make(chan *events.Envelope),
		errs:        make(chan error),
		heartbeater: heartbeater,
	}
	c := a.consumer()

	_, fhErrs := c.Start(p)
	defer c.Stop()

	//go func() {
	//	for err := range errs {
	//		a.logger.Error("nozzle", err)
	//	}
	//}()

	fatalErr := <-fhErrs
	if fatalErr != nil {
		a.logger.Fatal("firehose", fatalErr)
	}
}

type producer struct {
	messages chan *events.Envelope
	errs     chan error

	heartbeater heartbeat.Heartbeater
}

func (p *producer) Connect() (<-chan *events.Envelope, <-chan error) {
	p.heartbeater.Start()

	go func() {
		for {
			p.heartbeater.AddCounter()
			p.messages <- newEvent()
		}
	}()

	return p.messages, p.errs
}

func newEvent() *events.Envelope {
	eventType := eventTypes[rand.Intn(len(eventTypes))]
	return &events.Envelope{EventType: &eventType}
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
		MetricSink:  nil,
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

	metricBuffer, _ := stackdriver.NewMetricsBuffer(a.c.BatchCount, metricAdapter)
	//go func() {
	//	for err = range errs {
	//		a.logger.Error("metricsBuffer", err)
	//	}
	//}()

	return nozzle.NewMetricSink(a.labelMaker, metricBuffer, nozzle.NewUnitParser())
}
