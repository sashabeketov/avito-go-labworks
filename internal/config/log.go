package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type LogLevel string

// потом я добавлю допустимые уровни логирования, когда этап дойдет до логирования сервиса
const (
	LogLevelInfo = "info"
)

type LogConfig struct {
	LogLevel string `env:"LOG_LEVEL" env-required:"true" env-upd:"true" validate:"required,is-log-level"`
}

func loadLog() (LogConfig, error) {
	var logConfig LogConfig

	err := cleanenv.ReadEnv(&logConfig)

	if err != nil {
		return LogConfig{}, err
	}

	err = validate.RegisterValidation("is-log-level", isLogLevel)

	if err != nil {
		return LogConfig{}, err
	}

	err = validate.Struct(logConfig)

	if err != nil {
		return LogConfig{}, err
	}

	return logConfig, nil
}

func isLogLevel(fl validator.FieldLevel) bool {
	level := fl.Field().String()

	switch level {
	case LogLevelInfo:
		return true
	default:
		return false
	}
}
