package config

import (
	"github.com/go-playground/validator/v10"
)

type Config struct {
	HTTP     HTTPConfig
	Database DBConfig
	Log      LogConfig
	App      AppConfig
}

var validate *validator.Validate

func LoadConfig() (Config, error) {
	validate = validator.New()
	http, err := loadHTTP()

	if err != nil {
		return Config{}, err
	}

	database, err := loadDB()

	if err != nil {
		return Config{}, err
	}

	log, err := loadLog()

	if err != nil {
		return Config{}, err
	}

	app, err := loadApp()

	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP:     http,
		Database: database,
		Log:      log,
		App:      app,
	}, nil
}
