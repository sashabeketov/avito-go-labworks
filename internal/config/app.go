package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-required:"true" validate:"required,gt=0"`
}

func loadApp() (AppConfig, error) {
	var appConfig AppConfig

	err := cleanenv.ReadEnv(&appConfig)

	if err != nil {
		return AppConfig{}, err
	}

	err = validate.Struct(appConfig)

	if err != nil {
		return AppConfig{}, err
	}

	return appConfig, nil
}
