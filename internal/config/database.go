package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type DBConfig struct {
	DatabaseURL     string        `env:"DATABASE_URL" env-required:"true" validate:"required,url"`
	MaxConns        int           `env:"DATABASE_MAX_CONNS" env-required:"true" validate:"required,gt=0"`
	MinConns        int           `env:"DATABASE_MIN_CONNS" env-required:"true" validate:"required,gte=0"`
	MaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME" env-required:"true" validate:"required,gt=0"`
	ConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT" env-required:"true" validate:"required,gt=0"`
	QueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT" env-required:"true" validate:"required,gt=0"`
	PingTimeout     time.Duration `env:"DATABASE_PING_TIMEOUT" env-required:"true" validate:"required,gt=0"`
}

func loadDB() (DBConfig, error) {
	var dbConfig DBConfig

	err := cleanenv.ReadEnv(&dbConfig)

	if err != nil {
		return DBConfig{}, err
	}

	err = validate.Struct(dbConfig)

	if err != nil {
		return DBConfig{}, err
	}

	if dbConfig.MinConns > dbConfig.MaxConns {
		return DBConfig{}, fmt.Errorf("MaxConns must be greater than MinConns")
	}

	return dbConfig, nil
}
