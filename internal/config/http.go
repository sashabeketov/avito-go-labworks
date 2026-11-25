package config

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Addr              string        `env:"HTTP_ADDR" evn-required:"true" validate:"required,hostname_port"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" evn-required:"true" validate:"required,gt=0"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" evn-required:"true" validate:"required,gt=0"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" evn-required:"true" validate:"required,gt=0"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" evn-required:"true" validate:"required,gt=0"`
}

func loadHTTP(validate *validator.Validate) (HTTPConfig, error) {
	var httpConfig HTTPConfig

	err := cleanenv.ReadEnv(&httpConfig)
	if err != nil {
		return HTTPConfig{}, err
	}

	err = validate.Struct(httpConfig)
	if err != nil {
		return HTTPConfig{}, err
	}

	return httpConfig, nil
}
