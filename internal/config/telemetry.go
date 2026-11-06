package config

import (
	"fmt"
	"net/url"

	"github.com/ilyakaznacheev/cleanenv"
)

type TelemetryConfig struct {
	ServiceName string `env:"OTEL_SERVICE_NAME" env-required:"true"`
	Endpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" env-required:"true"`
	Protocol    string `env:"OTEL_EXPORTER_OTLP_PROTOCOL" env-required:"true"`
	Sampler     string `env:"OTEL_TRACES_SAMPLER" env-required:"true"`
}

func loadTelemetry() (TelemetryConfig, error) {
	var cfg TelemetryConfig
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return TelemetryConfig{}, err
	}

	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return TelemetryConfig{}, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute URL")
	}
	if cfg.Protocol != "http/protobuf" {
		return TelemetryConfig{}, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", cfg.Protocol)
	}
	switch cfg.Sampler {
	case "parentbased_always_on", "parentbased_always_off", "always_on", "always_off":
	default:
		return TelemetryConfig{}, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER %q", cfg.Sampler)
	}
	return cfg, nil
}
