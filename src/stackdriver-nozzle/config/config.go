/*
 * Copyright 2017 Google Inc.
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

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"cloud.google.com/go/compute/metadata"
	"code.cloudfoundry.org/lager"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

func NewConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}

	err = c.validate()
	if err != nil {
		return nil, err
	}

	err = c.ensureProjectID()
	if err != nil {
		return nil, err
	}

	err = c.maybeLoadFilterFile()
	if err != nil {
		return nil, err
	}

	c.setNozzleHostInfo()

	return &c, nil
}

type Config struct {
	// Firehose config
	APIEndpoint      string `envconfig:"firehose_endpoint" required:"true"`
	SubscriptionID   string `envconfig:"firehose_subscription_id" required:"false"`
	LoggingEvents    string `envconfig:"firehose_events_to_stackdriver_logging" required:"true"`
	MonitoringEvents string `envconfig:"firehose_events_to_stackdriver_monitoring" required:"false"`
	Username         string `envconfig:"firehose_username" default:"admin"`
	Password         string `envconfig:"firehose_password" default:"admin"`
	SkipSSL          bool   `envconfig:"firehose_skip_ssl" default:"false"`
	NewlineToken     string `envconfig:"firehose_newline_token"`

	// Reverse Log Proxy (Firehose alternative) config
	//TODO(evanbrown): Determine which flags should be required
	RLPAddress           string `envconfig:"rlp_address_colon_port" required:"false"`
  RLPCACertFile        string `envconfig:"rlp_ca_cert_file" required:"false"`
  RLPCertFile          string `envconfig:"rlp_cert_file" required:"false"`
  RLPKeyFile           string `envconfig:"rlp_key_file" required:"false"`
	RLPShardID           string `envconfig:"rlp_shard_id" default:"stackdriver-nozzle"`
	RLPDeterministicName string `envconfig:"rlp_deterministic_name"`

	// Stackdriver config
	ProjectID            string `envconfig:"gcp_project_id"`
	LoggingBatchCount    int    `envconfig:"logging_batch_count" default:"1000"`
	LoggingBatchDuration int    `envconfig:"logging_batch_duration" default:"30"`
	LoggingReqsInFlight  int    `envconfig:"logging_requests_in_flight" default:"16"`

	// Nozzle config
	HeartbeatRate         int    `envconfig:"heartbeat_rate" default:"30"`
	MetricsBufferDuration int    `envconfig:"metrics_buffer_duration" default:"30"`
	MetricsBatchSize      int    `envconfig:"metrics_batch_size" default:"200"`
	MetricPathPrefix      string `envconfig:"metric_path_prefix" default:"firehose"`
	FoundationName        string `envconfig:"foundation_name" default:"cf"`
	ResolveAppMetadata    bool   `envconfig:"resolve_app_metadata"`
	NozzleID              string `envconfig:"nozzle_id" default:"local-nozzle"`
	NozzleName            string `envconfig:"nozzle_name" default:"local-nozzle"`
	NozzleZone            string `envconfig:"nozzle_zone" default:"local-nozzle"`
	DebugNozzle           bool   `envconfig:"debug_nozzle"`
	// By default 'origin' label is prepended to metric name, however for runtime metrics (defined here) we add it as a metric label instead.
	RuntimeMetricRegex string `envconfig:"runtime_metric_regex" default:"^(numCPUS|numGoRoutines|memoryStats\\..*)$"`
	// If enabled, CounterEvents will be reported as cumulative Stackdriver metrics instead of two gauges (<metric>.delta
	// and <metric>.total). Reporting cumulative metrics involves nozzle keeping track of internal counter state, and
	// requires deterministic routing of CounterEvents to nozzles (i.e. CounterEvent messages for a particular metric MUST
	// always be routed to the same nozzle process); the easiest way to achieve that is to run a single copy of the nozzle.
	EnableCumulativeCounters bool `envconfig:"enable_cumulative_counters"`
	// If enabled, the Nozzle will derive per-application HTTP metrics from
	// HttpStartStop events and export them as counters to Stackdriver.
	EnableAppHTTPMetrics bool `envconfig:"enable_app_http_metrics"`
	// Expire internal counter state if a given counter has not been seen for this many seconds.
	CounterTrackerTTL int `envconfig:"counter_tracker_ttl" default:"130"`

	// Event blacklists / whitelists are too complex to stuff into environment
	// vars, so instead they are templated from the manifest YAML into a JSON
	// file which is loaded by the nozzle. Nil pointers are empty blacklists.
	EventFilterFile string `envconfig:"event_filter_file" default:""`
	EventFilterJSON *EventFilterJSON
}

//TODO(evanbrown): Validate configs for both Firehose and RLP modes
func (c *Config) validate() error {
	if c.APIEndpoint == "" {
		return errors.New("FIREHOSE_ENDPOINT is empty")
	}

	if c.LoggingEvents == "" && c.MonitoringEvents == "" {
		return errors.New("FIREHOSE_EVENTS_TO_STACKDRIVER_LOGGING and FIREHOSE_EVENTS_TO_STACKDRIVER_MONITORING are empty")
	}

	return nil
}

func (c *Config) ensureProjectID() error {
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

// An EventFilterRule specifies a filtering rule for firehose event.
type EventFilterRule struct {
	// Must be one of the types from nozzle/event_filter.go.
	Type string `json:"type"`
	// Must be either "monitoring", "logging", or "all".
	Sink string `json:"sink"`
	// Must be a valid regular expression.
	Regexp string `json:"regexp"`
}

func (r EventFilterRule) String() string {
	return fmt.Sprintf("%s.%s matches %q", r.Sink, r.Type, r.Regexp)
}

type EventFilterJSON struct {
	Blacklist []EventFilterRule `json:"blacklist,omitempty"`
	Whitelist []EventFilterRule `json:"whitelist,omitempty"`
}

func (c *Config) maybeLoadFilterFile() error {
	if c.EventFilterFile == "" {
		return nil
	}
	fh, err := os.Open(c.EventFilterFile)
	if err != nil {
		return err
	}

	if err := c.parseEventFilterJSON(fh); err != nil {
		return err
	}

	return fh.Close()
}

func (c *Config) parseEventFilterJSON(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		// Unmarshal expects there to be at least some JSON ;-)
		return nil
	}
	c.EventFilterJSON = &EventFilterJSON{}
	return json.Unmarshal(data, c.EventFilterJSON)
}

// If running on GCE, this will set the nozzle's ID, name, and zone to
// the GCE instance's values.
func (c *Config) setNozzleHostInfo() {
	if metadata.OnGCE() {
		if v, err := metadata.InstanceID(); err == nil {
			c.NozzleID = v
		}
		if v, err := metadata.Zone(); err == nil {
			c.NozzleZone = v
		}
		if v, err := metadata.InstanceName(); err == nil {
			c.NozzleName = v
		}
	}
}

func (c *Config) ToData() lager.Data {
	return lager.Data{
		"APIEndpoint":                   c.APIEndpoint,
		"Username":                      c.Username,
		"Password":                      "<redacted>",
		"EventsToStackdriverMonitoring": c.MonitoringEvents,
		"EventsToStackdriverLogging":    c.LoggingEvents,
		"SkipSSL":                       c.SkipSSL,
		"ProjectID":                     c.ProjectID,
		"LoggingBatchCount":             c.LoggingBatchCount,
		"LoggingBatchDuration":          c.LoggingBatchDuration,
		"HeartbeatRate":                 c.HeartbeatRate,
		"ResolveAppMetadata":            c.ResolveAppMetadata,
		"SubscriptionID":                c.SubscriptionID,
		"DebugNozzle":                   c.DebugNozzle,
		"NewlineToken":                  c.NewlineToken,
	}
}
