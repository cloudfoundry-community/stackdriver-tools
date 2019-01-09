/*
 * Copyright 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/config"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/metricspipeline"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"github.com/cloudfoundry/sonde-go/events"
)

type App struct {
	logger      lager.Logger
	c           *config.Config
	cfConfig    *cfclient.Config
	cfClient    *cfclient.Client
	rlpConfig   *cloudfoundry.ReverseLogProxyConfig
	labelMaker  nozzle.LabelMaker
	bufferEmpty func() bool
}

func New(c *config.Config, logger lager.Logger) *App {
	logger.Info("version", lager.Data{"name": version.Name, "release": version.Release(), "user_agent": version.UserAgent()})
	logger.Info("arguments", c.ToData())

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
	labelMaker := nozzle.NewLabelMaker(appInfoRepository, c.FoundationName)

	tlsConfig, err := loggregator.NewEgressTLSConfig(
		c.RLPCACertFile,
		c.RLPCertFile,
		c.RLPKeyFile,
	)
	if err != nil {
		logger.Fatal("could not create TLS config", err)
	}

	rlpConfig := &cloudfoundry.ReverseLogProxyConfig{
		Address:           c.RLPAddress,
		ShardID:           c.RLPShardID,
		DeterministicName: c.RLPDeterministicName,
		TLSConfig:         tlsConfig,
	}

	return &App{
		logger:     logger,
		c:          c,
		cfConfig:   cfConfig,
		cfClient:   cfClient,
		rlpConfig:  rlpConfig,
		labelMaker: labelMaker,
	}
}

func (a *App) newProducer() cloudfoundry.ReverseLogProxy {
	return cloudfoundry.NewReverseLogProxy(a.rlpConfig, a.logger)
}

func (a *App) newConsumer(ctx context.Context) (nozzle.Nozzle, error) {
	logEvents, err := nozzle.ParseEvents(strings.Split(a.c.LoggingEvents, ","))
	if err != nil {
		return nil, err
	}

	metricEvents, err := nozzle.ParseEvents(strings.Split(a.c.MonitoringEvents, ","))
	if err != nil {
		return nil, err
	}

	lbl, lwl, mbl, mwl, err := a.buildEventFilters()
	if err != nil {
		return nil, err
	}

	var sinks []nozzle.Sink
	logAdapter := a.newLogAdapter()
	filteredLogSink, err := nozzle.NewFilterSink(logEvents, lbl, lwl,
		nozzle.NewLogSink(a.labelMaker, logAdapter, a.c.NewlineToken, a.logger))
	if err != nil {
		return nil, err
	}
	sinks = append(sinks, filteredLogSink)

	// Destination for metrics
	metricAdapter := a.newMetricAdapter()
	// Routes metrics to Stackdriver Logging/Stackdriver Monitoring
	metricRouter := metricspipeline.NewRouter(metricAdapter, metricEvents, logAdapter, logEvents)
	// Handles and translates Firehose events. Performs buffering/culling.
	metricSink, err := a.newMetricSink(ctx, metricRouter)
	if err != nil {
		return nil, err
	}
	// Filter Firehose events to what the user selects
	filteredMetricSink, err := nozzle.NewFilterSink(metricEvents, mbl, mwl, metricSink)
	if err != nil {
		return nil, err
	}
	sinks = append(sinks, filteredMetricSink)

	if a.c.EnableAppHTTPMetrics {
		httpSink := nozzle.NewHTTPSink(a.logger, a.labelMaker)
		filteredHTTPSink, err := nozzle.NewFilterSink([]events.Envelope_EventType{events.Envelope_HttpStartStop}, nil, nil, httpSink)
		if err != nil {
			return nil, err
		}
		sinks = append(sinks, filteredHTTPSink)
	}

	return nozzle.NewNozzle(a.logger, sinks...), nil
}

func (a *App) newLogAdapter() stackdriver.LogAdapter {
	logAdapter, logErrs := stackdriver.NewLogAdapter(
		a.c.ProjectID,
		a.c.LoggingBatchCount,
		time.Duration(a.c.LoggingBatchDuration)*time.Second,
		a.c.LoggingReqsInFlight,
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

	metricAdapter, err := stackdriver.NewMetricAdapter(a.c.ProjectID, metricClient, a.c.MetricsBatchSize, a.logger)
	if err != nil {
		a.logger.Fatal("metricAdapter", err)
	}

	return metricAdapter
}

func (a *App) newMetricSink(ctx context.Context, metricAdapter stackdriver.MetricAdapter) (nozzle.Sink, error) {
	metricBuffer := metricspipeline.NewAutoCulledMetricsBuffer(ctx, a.logger, time.Duration(a.c.MetricsBufferDuration)*time.Second, metricAdapter)
	a.bufferEmpty = metricBuffer.IsEmpty

	var counterTracker *nozzle.CounterTracker
	if a.c.EnableCumulativeCounters {
		ttl := time.Duration(a.c.CounterTrackerTTL) * time.Second
		counterTracker = nozzle.NewCounterTracker(ctx, ttl, a.logger)
	}

	return nozzle.NewMetricSink(a.logger, a.c.MetricPathPrefix, a.labelMaker, metricBuffer, counterTracker, nozzle.NewUnitParser(), a.c.RuntimeMetricRegex)
}

func (a *App) newTelemetryReporter() telemetry.Reporter {
	metricClient, err := stackdriver.NewMetricClient()
	if err != nil {
		a.logger.Fatal("metricClient", err)
	}

	logSink := telemetry.NewLogSink(a.logger)
	metricSink := stackdriver.NewTelemetrySink(a.logger, metricClient, a.c.ProjectID, a.c.SubscriptionID, a.c.FoundationName)
	return telemetry.NewReporter(time.Duration(a.c.HeartbeatRate)*time.Second, logSink, metricSink)
}

var validSinks = map[string]bool{"monitoring": true, "logging": true, "all": true}

func (a *App) buildEventFilters() (
	loggingBlacklist *nozzle.EventFilter,
	loggingWhitelist *nozzle.EventFilter,
	monitoringBlacklist *nozzle.EventFilter,
	monitoringWhitelist *nozzle.EventFilter,
	err error,
) {
	var errs []error
	if len(a.c.EventFilterJSON.Blacklist) > 0 {
		loggingBlacklist = &nozzle.EventFilter{}
		monitoringBlacklist = &nozzle.EventFilter{}
		errs = append(errs, loadFilterRules(a.c.EventFilterJSON.Blacklist, loggingBlacklist, monitoringBlacklist)...)
	}
	if len(a.c.EventFilterJSON.Whitelist) > 0 {
		loggingWhitelist = &nozzle.EventFilter{}
		monitoringWhitelist = &nozzle.EventFilter{}
		errs = append(errs, loadFilterRules(a.c.EventFilterJSON.Whitelist, loggingWhitelist, monitoringWhitelist)...)
	}
	if len(errs) == 0 {
		return loggingBlacklist, loggingWhitelist, monitoringBlacklist, monitoringWhitelist, nil
	}

	b := bytes.NewBufferString("encountered the following errors while building event filters:")
	for _, err := range errs {
		b.WriteString("\n\t- ")
		b.WriteString(err.Error())
	}
	b.WriteByte('\n')
	return nil, nil, nil, nil, errors.New(b.String())
}

func loadFilterRules(list []config.EventFilterRule, loggingFilter, monitoringFilter *nozzle.EventFilter) []error {
	var errs []error
	for _, rule := range list {
		if !validSinks[rule.Sink] {
			errs = append(errs, fmt.Errorf("rule %s has invalid sink %q", rule, rule.Sink))
			continue
		}
		if rule.Regexp == "" {
			errs = append(errs, fmt.Errorf("rule %s has empty regexp", rule))
			continue
		}
		if rule.Sink == "monitoring" || rule.Sink == "all" {
			if err := monitoringFilter.Add(rule.Type, rule.Regexp); err != nil {
				errs = append(errs, fmt.Errorf("adding rule %s to monitoring filter failed: %v", rule, err))
			}
		}
		if rule.Sink == "logging" || rule.Sink == "all" {
			if err := loggingFilter.Add(rule.Type, rule.Regexp); err != nil {
				errs = append(errs, fmt.Errorf("adding rule %s to logging filter failed: %v", rule, err))
			}
		}
	}
	return errs
}
