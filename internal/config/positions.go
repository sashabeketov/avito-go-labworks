package config

import (
	"errors"
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type PositionsConfig struct {
	CheckInterval time.Duration `env:"POSITION_CHECK_INTERVAL" env-default:"30s" validate:"gt=0"`
	StaleAfter    time.Duration `env:"POSITION_STALE_AFTER" env-default:"60s" validate:"gt=0"`
	Workers       int           `env:"POSITION_WORKERS" env-default:"4" validate:"gt=0"`
	QueueSize     int           `env:"POSITION_QUEUE_SIZE" env-default:"100" validate:"gt=0"`
	PushHTTPURL   string        `env:"PUSH_HTTP_URL" env-required:"true"`
	PushTimeout   time.Duration `env:"PUSH_REQUEST_TIMEOUT" env-default:"1s" validate:"gt=0"`
}

func loadPositions(validate *validator.Validate) (PositionsConfig, error) {
	var cfg PositionsConfig
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return PositionsConfig{}, err
	}
	if err := validate.Struct(cfg); err != nil {
		return PositionsConfig{}, err
	}
	endpoint, err := url.Parse(cfg.PushHTTPURL)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return PositionsConfig{}, errors.New("PUSH_HTTP_URL must be an absolute HTTP or HTTPS URL")
	}
	return cfg, nil
}
