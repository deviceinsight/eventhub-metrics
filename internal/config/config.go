package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

type NamespaceConfig struct {
	Endpoint               string
	StorageAccountEndpoint string
	CheckpointContainer    string
}

type BlobStorageConfig struct {
	Name     string
	Endpoint string
}

type AppInsightsConfig struct {
	Enabled bool
}

type PrometheusConfig struct {
	Enabled     bool
	ReadTimeout time.Duration
	Address     string
}

type ExporterConfig struct {
	AppInsights AppInsightsConfig
	Prometheus  PrometheusConfig
}

type CollectorConfig struct {
	OwnershipExpirationDuration time.Duration
	Concurrency                 int
	Interval                    *time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

type Config struct {
	Namespaces []NamespaceConfig
	Exporter   ExporterConfig
	Collector  CollectorConfig
	Log        LogConfig
}

const EnvPrefix string = "EH_METRICS_"

func Load(path string) (*Config, error) {

	var k = koanf.New(".")

	// load defaults
	if err := k.Load(confmap.Provider(map[string]interface{}{
		"log.level":                             "INFO",
		"log.format":                            "JSON",
		"collector.ownershipExpirationDuration": time.Minute,
		"collector.concurrency":                 8, //nolint:mnd // just a default
		"exporter.prometheus.address":           ":8080",
		"exporter.prometheus.readTimeout":       "1s",
	}, "."), nil); err != nil {
		return nil, fmt.Errorf("failed to load config defaults: %w", err)
	}

	// load config file
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// load env variables
	if err := k.Load(env.Provider(EnvPrefix, ".", func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(s, EnvPrefix), "_", "."))
	}), nil); err != nil {
		return nil, fmt.Errorf("failed read env variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
