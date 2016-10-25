package config

import (
	"cloud.google.com/go/compute/metadata"
	"github.com/cloudfoundry/lager"
	"github.com/kelseyhightower/envconfig"
)

func NewConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}

	err = c.ensureProjectID()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

type Config struct {
	// Firehose config
	APIEndpoint    string `envconfig:"firehose_endpoint" required:"true"`
	Events         string `envconfig:"firehose_events" required:"true"`
	Username       string `envconfig:"firehose_username" default:"admin"`
	Password       string `envconfig:"firehose_password" default:"admin"`
	SkipSSL        bool   `envconfig:"firehose_skip_ssl" default:"false"`
	SubscriptionID string `envconfig:"firehose_subscription_id"`

	// Stackdriver config
	ProjectID string `envconfig:"gcp_project_id"`

	// Nozzle config
	HeartbeatRate      int    `envconfig:"heartbeat_rate" default:"30"`
	BatchCount         int    `envconfig:"batch_count" default:"10"`
	BatchDuration      int    `envconfig:"batch_duration" default:"1"`
	BoltDBPath         string `envconfig:"boltdb_path" default:"cached-app-metadata.db"`
	ResolveAppMetadata bool   `envconfig:"resolve_app_metadata"`
	DebugNozzle        bool   `envconfig:"debug_nozzle"`
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

func (c *Config) ToData() lager.Data {
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
		"DebugNozzle":        c.DebugNozzle,
	}
}
