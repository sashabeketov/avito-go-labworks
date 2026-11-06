package config

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type AppConfig struct {
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-required:"true" validate:"required,gt=0"`
}

func loadApp(validate *validator.Validate) (AppConfig, error) {
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
