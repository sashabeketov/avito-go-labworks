package config

import (
	"time"
)

type AppConfig struct {
	ShutdownTimeout time.Duration
}

func loadApp() (AppConfig, error) {
	val, err := requireDuration("SHUTDOWN_TIMEOUT")

	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		ShutdownTimeout: val,
	}, nil
}
