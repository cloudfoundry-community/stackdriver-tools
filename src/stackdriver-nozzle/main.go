package main

import (
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/lager"
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	// Firehose config
	APIEndpoint string `envconfig:"firehose_endpoint" required:"true"`
	Events      string `envconfig:"firehose_events" required:"true"`
	Username    string `envconfig:"firehose_username" default:"admin"`
	Password    string `envconfig:"firehose_password" default:"admin"`
	SkipSSL     bool   `envconfig:"firehose_skip_ssl" default:"false"`

	// Stackdriver config
	ProjectID string `envconfig:"gcp_project_id"`

	// Nozzle config
	HeartbeatRate      int    `envconfig:"heartbeat_rate" default:"30"`
	BatchCount         int    `envconfig:"batch_count" default:"10"`
	BatchDuration      int    `envconfig:"batch_duration" default:"1"`
	BoltDBPath         string `envconfig:"boltdb_path" default:"cached-app-metadata.db"`
	ResolveAppMetadata bool   `envconfig:"resolve_app_metadata" default:"true"`
	SubscriptionID     string `envconfig:"subscription_id" default:"stackdriver-nozzle"`
}

func (c *config) ensureProjectID() error {
	if c.ProjectID != "" {
		return nil
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		return err
	}

	c.ProjectID = projectID
	return nil
}

func (c *config) toData() lager.Data {
	return lager.Data{
		"APIEndpoint":        c.APIEndpoint,
		"Username":           c.Username,
		"Password":           "<redacted>",
		"Events":             c.Events,
		"SkipSSL":            c.SkipSSL,
		"ProjectID":          c.ProjectID,
		"BatchCount":         c.BatchCount,
		"BatchDuration":      c.BatchDuration,
		"HeartbeatRate":      c.HeartbeatRate,
		"BoltDBPath":         c.BoltDBPath,
		"ResolveAppMetadata": c.ResolveAppMetadata,
		"SubscriptionID":     c.SubscriptionID,
	}
}

func main() {
	logger := lager.NewLogger("stackdriver-nozzle")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	var c config
	err := envconfig.Process("", &c)
	if err != nil {
		logger.Fatal("envconfig", err)
	}

	err = c.ensureProjectID()
	if err != nil {
		logger.Fatal("gcpProjectID", err)
	}

	logger.Info("arguments", c.toData())

	cfConfig := &cfclient.Config{
		ApiAddress:        c.APIEndpoint,
		Username:          c.Username,
		Password:          c.Password,
		SkipSslValidation: c.SkipSSL}
	cfClient := cfclient.NewClient(cfConfig)
	input := firehose.NewClient(cfConfig, cfClient, logger, c.SubscriptionID)

	var cachingClient caching.Caching
	if c.ResolveAppMetadata {
		cachingClient = caching.NewCachingBolt(cfClient, c.BoltDBPath)
	} else {
		cachingClient = caching.NewCachingEmpty()
	}
	cachingClient.CreateBucket()

	logAdapter, err := stackdriver.NewLogAdapter(
		c.ProjectID,
		c.BatchCount,
		time.Duration(c.BatchDuration)*time.Second,
		logger,
	)
	if err != nil {
		logger.Fatal("newLogAdapter", err)
	}

	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		logger.Fatal("newMetricClient", err)
	}

	metricAdapter, err := stackdriver.NewMetricAdapter(c.ProjectID, metricClient)
	if err != nil {
		logger.Fatal("newMetricAdapter", err)
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

	output := nozzle.Nozzle{
		LogHandler:    logHandler,
		MetricHandler: metricHandler,
		Heartbeater:   heartbeater,
	}

	filteredOutput, err := filter.New(&output, strings.Split(c.Events, ","))
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
