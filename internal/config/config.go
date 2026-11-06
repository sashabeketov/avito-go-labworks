package config

import (
	"github.com/go-playground/validator/v10"
)

type Config struct {
	HTTP      HTTPConfig
	Database  DBConfig
	Log       LogConfig
	App       AppConfig
	Telemetry TelemetryConfig
}

func LoadConfig() (Config, error) {
	validate := validator.New()
	http, err := loadHTTP(validate)
	if err != nil {
		return Config{}, err
	}

	database, err := loadDB(validate)
	if err != nil {
		return Config{}, err
	}

	log, err := loadLog(validate)
	if err != nil {
		return Config{}, err
	}

	app, err := loadApp(validate)
	if err != nil {
		return Config{}, err
	}

	telemetry, err := loadTelemetry()
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP:      http,
		Database:  database,
		Log:       log,
		App:       app,
		Telemetry: telemetry,
	}, nil
}
